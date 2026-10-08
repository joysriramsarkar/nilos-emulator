package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyTomlFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nilemu.toml")
	content := `
# nilemu configuration
listen_addr = "127.0.0.1"
data_dir = "/tmp/nilemu-data"
nil_bin = "/usr/local/bin/nil"
allow_unsigned = true

[server]
port = 9111

[keys]
test-key = "abcdef"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := applyTomlFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9111 {
		t.Errorf("port = %d, want 9111", cfg.Port)
	}
	if cfg.ListenAddr != "127.0.0.1" {
		t.Errorf("addr = %s", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/nilemu-data" {
		t.Errorf("data dir = %s", cfg.DataDir)
	}
	if cfg.NilBin != "/usr/local/bin/nil" {
		t.Errorf("nil bin = %s", cfg.NilBin)
	}
	if !cfg.AllowUnsigned {
		t.Error("allow_unsigned not parsed")
	}
	if cfg.TrustedKeys["test-key"] != "abcdef" {
		t.Errorf("trusted key not parsed: %+v", cfg.TrustedKeys)
	}
}

func TestDefaultIsLoopback(t *testing.T) {
	cfg := Default()
	if cfg.ListenAddr != "127.0.0.1" {
		t.Fatalf("default must bind loopback, got %s", cfg.ListenAddr)
	}
	if cfg.LoopbackAddr() == "" {
		t.Fatal("empty loopback addr")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("NILEMU_PORT", "9555")
	t.Setenv("NILEMU_ALLOW_UNSIGNED", "true")
	cfg := Default()
	applyEnv(&cfg)
	if cfg.Port != 9555 {
		t.Errorf("port = %d, want 9555", cfg.Port)
	}
	if !cfg.AllowUnsigned {
		t.Error("expected AllowUnsigned from env")
	}
}
