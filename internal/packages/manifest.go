package packages

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var appIDRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)

// Limits caps an app's resource usage.
type Limits struct {
	MemoryMB int `json:"memory_mb,omitempty"`
	CPUPct   int `json:"cpu_pct,omitempty"`
}

// Manifest is the metadata file (manifest.json) at the root of a .nilax
// bundle.
type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Entry       string   `json:"entry"`
	Icon        string   `json:"icon,omitempty"`
	Author      string   `json:"author,omitempty"`
	MinRuntime  string   `json:"min_runtime,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Devices     []string `json:"devices,omitempty"`
	Limits      *Limits  `json:"limits,omitempty"`
}

// ParseManifest decodes and validates manifest bytes.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		// Retry leniently so forward-compatible unknown fields do not break
		// older emulator versions.
		if jerr := json.Unmarshal(data, &m); jerr != nil {
			return nil, fmt.Errorf("invalid manifest json: %w", jerr)
		}
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks required fields and path conventions.
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return errors.New("manifest.id is required")
	}
	if !appIDRegex.MatchString(m.ID) {
		return fmt.Errorf("manifest.id is not a valid reverse-domain id: %s", m.ID)
	}
	if m.Name == "" {
		return errors.New("manifest.name is required")
	}
	if m.Version == "" {
		return errors.New("manifest.version is required")
	}
	if m.Entry == "" {
		return errors.New("manifest.entry is required")
	}
	if !strings.HasPrefix(m.Entry, "payload/") {
		return errors.New("manifest.entry must be inside payload/")
	}
	if m.Icon != "" && !strings.HasPrefix(m.Icon, "payload/") {
		return errors.New("manifest.icon must be inside payload/ if provided")
	}
	seen := map[string]bool{}
	for _, p := range m.Permissions {
		p = strings.TrimSpace(p)
		if p == "" {
			return errors.New("empty permission")
		}
		if seen[p] {
			return fmt.Errorf("duplicate permission: %s", p)
		}
		seen[p] = true
	}
	return nil
}

// PermissionsSet returns the permission list as a lowercase set.
func (m *Manifest) PermissionsSet() map[string]bool {
	out := make(map[string]bool, len(m.Permissions))
	for _, p := range m.Permissions {
		out[strings.ToLower(strings.TrimSpace(p))] = true
	}
	return out
}
