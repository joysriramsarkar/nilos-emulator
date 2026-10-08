// Package config loads emulator configuration from an optional nilemu.toml
// file and environment variables. It intentionally has no external
// dependencies so that nilemu can remain a single static binary.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Config holds every tunable setting for the emulator host.
type Config struct {
	// ListenAddr is the TCP address the control/HTTP API binds to.
	// It defaults to 127.0.0.1 (loopback only) for security.
	ListenAddr string
	// Port is the HTTP control API port.
	Port int
	// DataDir is the root of the persistent emulator state
	// (devices, snapshots, logs). Defaults to ~/.nilemu.
	DataDir string
	// NilBin is the path to the NilLang `nil` binary used for the
	// optional real runtime mode. May be empty when the binary is not
	// installed; the emulator then falls back to the built-in app host.
	NilBin string
	// NilosdBin is the path to the runtime worker binary. Empty means
	// "sibling of the current executable".
	NilosdBin string
	// AuthToken is a per-session random token required by mutating API
	// calls. Empty disables token enforcement (only for tests).
	AuthToken string
	// FirmwareDir is the directory containing *.nilax sample bundles and
	// system apps that are installed into a fresh device.
	FirmwareDir string
	// TrustedKeys maps key ids to hex-encoded Ed25519 public keys used to
	// verify .nilax bundles.
	TrustedKeys map[string]string
	// AllowUnsigned permits installing unsigned bundles (developer mode).
	AllowUnsigned bool
	// Headless disables the auto-open of the browser and any GUI.
	Headless bool
	// ClockISO, when set, initialises the virtual clock to this RFC3339 time.
	ClockISO string
	// ClockFrozen stops the virtual clock so runs are reproducible.
	ClockFrozen bool
	// Seed seeds deterministic RNG in the runtime.
	Seed int64
	// Animations enables UI animation timing (disabled for golden tests).
	Animations bool
}

// Default returns the built-in defaults before any file/env override.
func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return Config{
		ListenAddr:  "127.0.0.1",
		Port:        8085,
		DataDir:     filepath.Join(home, ".nilemu"),
		NilBin:      "",
		NilosdBin:   "",
		FirmwareDir: "firmware",
		TrustedKeys: map[string]string{},
		Animations:  true,
	}
}

// LoopbackAddr returns the host:port string for the control API.
func (c Config) LoopbackAddr() string {
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.Port)
}

// Load reads nilemu.toml (if present) then applies environment overrides.
// Search order for the file: $NILEMU_CONFIG, ./nilemu.toml,
// $DataDir/nilemu.toml.
func Load() (Config, error) {
	cfg := Default()

	path := os.Getenv("NILEMU_CONFIG")
	if path == "" {
		if _, err := os.Stat("nilemu.toml"); err == nil {
			path = "nilemu.toml"
		}
	}
	if path != "" {
		if err := applyTomlFile(&cfg, path); err != nil {
			return cfg, fmt.Errorf("config: %w", err)
		}
	}

	applyEnv(&cfg)
	cfg.DataDir = expandHome(cfg.DataDir)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("NILEMU_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("NILEMU_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("NILEMU_DATA"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("NIL_BIN"); v != "" {
		cfg.NilBin = v
	}
	if v := os.Getenv("NILOSD_BIN"); v != "" {
		cfg.NilosdBin = v
	}
	if v := os.Getenv("NILEMU_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
	if v := os.Getenv("NILEMU_FIRMWARE"); v != "" {
		cfg.FirmwareDir = v
	}
	if strings.EqualFold(os.Getenv("NILEMU_ALLOW_UNSIGNED"), "1") ||
		strings.EqualFold(os.Getenv("NILEMU_ALLOW_UNSIGNED"), "true") {
		cfg.AllowUnsigned = true
	}
	if strings.EqualFold(os.Getenv("NILEMU_HEADLESS"), "1") {
		cfg.Headless = true
	}
	if v := os.Getenv("NILEMU_CLOCK_ISO"); v != "" {
		cfg.ClockISO = v
	}
	if strings.EqualFold(os.Getenv("NILEMU_CLOCK_FROZEN"), "1") ||
		strings.EqualFold(os.Getenv("NILEMU_CLOCK_FROZEN"), "true") {
		cfg.ClockFrozen = true
	}
	if v := os.Getenv("NILEMU_SEED"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Seed = n
		}
	}
	if v := os.Getenv("NILEMU_ANIMATIONS"); strings.EqualFold(v, "off") {
		cfg.Animations = false
	}
}

// applyTomlFile implements a deliberately tiny TOML subset: comments with
// '#', [sections], and `key = value` pairs with string, int, bool, or
// array-of-string values. This is enough for nilemu.toml and avoids a
// dependency on a full TOML parser.
func applyTomlFile(cfg *Config, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	section := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNo)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if i := strings.IndexByte(val, '#'); i >= 0 && !strings.Contains(val[:i], "\"") {
			val = strings.TrimSpace(val[:i])
		}
		if err := setKey(cfg, section, key, unquote(val)); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
	}
	return sc.Err()
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func setKey(cfg *Config, section, key, val string) error {
	full := key
	if section != "" {
		full = section + "." + key
	}
	switch full {
	case "server.addr", "listen_addr":
		cfg.ListenAddr = val
	case "server.port", "port":
		p, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid port %q", val)
		}
		cfg.Port = p
	case "data_dir":
		cfg.DataDir = val
	case "nil_bin":
		cfg.NilBin = val
	case "nilosd_bin":
		cfg.NilosdBin = val
	case "firmware_dir":
		cfg.FirmwareDir = val
	case "allow_unsigned":
		cfg.AllowUnsigned = isTrue(val)
	case "clock_iso":
		cfg.ClockISO = val
	case "clock_frozen":
		cfg.ClockFrozen = isTrue(val)
	case "seed":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.Seed = n
		}
	case "animations":
		cfg.Animations = isTrue(val)
	case "keys":
		// keys.<keyid> = "<hex pubkey>"  (keys section)
		if section == "keys" {
			cfg.TrustedKeys[key] = val
		}
	}
	// key id under [keys]
	if section == "keys" {
		cfg.TrustedKeys[key] = val
	}
	return nil
}

func isTrue(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(p, `~\`)) {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// ErrNoConfig is returned by ReadFile when the requested file is missing.
var ErrNoConfig = errors.New("config file not found")
