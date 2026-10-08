package supervisor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
)

// DebugFrame is a stack frame reported to the DAP adapter.
type DebugFrame struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// DebugVariable is a name/value pair.
type DebugVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// DebugScope groups variables.
type DebugScope struct {
	Name               string `json:"name"`
	VariablesReference int    `json:"variablesReference"`
	Expensive          bool   `json:"expensive"`
}

// DebugBreakpoint is a breakpoint record.
type DebugBreakpoint struct {
	ID       int    `json:"id"`
	Verified bool   `json:"verified"`
	Line     int    `json:"line"`
	Source   string `json:"source"`
}

// DebugStack returns a stack trace reflecting the live session. Without the
// real NilLang bytecode debug info we cannot map to source lines, so we report
// the session/worker frames and mark the line as unknown (0) rather than
// inventing a location.
func (s *Supervisor) DebugStack(deviceID, appID string) []DebugFrame {
	se := s.get(deviceID, appID)
	if se == nil {
		return nil
	}
	frames, patches := se.sess.Stats()
	frames2 := []DebugFrame{
		{
			ID: 1, Name: fmt.Sprintf("%s (app worker)", appID),
			Source: se.app.Entry, Line: 0, Column: 0,
		},
		{
			ID: 2, Name: "NilHost.session",
			Source: "docs/protocol.md", Line: 0, Column: 0,
		},
	}
	_ = frames
	_ = patches
	return frames2
}

// DebugScopes returns the variable groups.
func (s *Supervisor) DebugScopes(deviceID, appID string) []DebugScope {
	return []DebugScope{
		{Name: "Session", VariablesReference: 1},
		{Name: "Device", VariablesReference: 2},
	}
}

// DebugVariables returns real session/device values.
func (s *Supervisor) DebugVariables(deviceID, appID string, ref int) []DebugVariable {
	se := s.get(deviceID, appID)
	switch ref {
	case 1:
		if se == nil {
			return []DebugVariable{{Name: "state", Value: "stopped"}}
		}
		frames, patches := se.sess.Stats()
		out := []DebugVariable{
			{Name: "session", Value: se.sess.ID, Type: "string"},
			{Name: "state", Value: se.sess.State(), Type: "string"},
			{Name: "frames", Value: fmt.Sprintf("%d", frames), Type: "uint64"},
			{Name: "patches", Value: fmt.Sprintf("%d", patches), Type: "uint64"},
			{Name: "pid", Value: fmt.Sprintf("%d", se.sess.Pid()), Type: "int"},
		}
		if saved := se.sess.SavedState(); len(saved) > 0 {
			out = append(out, DebugVariable{Name: "saved_state", Value: truncate(string(saved), 120), Type: "json"})
		}
		return out
	case 2:
		dev, err := s.manager.GetDevice(deviceID)
		if err != nil {
			return nil
		}
		settings, profileID := dev.SettingsSnapshot()
		return []DebugVariable{
			{Name: "id", Value: deviceID, Type: "string"},
			{Name: "profile", Value: profileID, Type: "string"},
			{Name: "locale", Value: settings.Locale, Type: "string"},
			{Name: "theme", Value: settings.Theme, Type: "string"},
			{Name: "battery", Value: fmt.Sprintf("%d%%", settings.BatteryLevel), Type: "int"},
		}
	}
	return nil
}

// SetBreakpoints records breakpoints (forwarded to the VM if it supports debug
// hooks). The mock/standalone run verifies them syntactically.
func (s *Supervisor) SetBreakpoints(deviceID, appID, source string, lines []int) []DebugBreakpoint {
	se := s.get(deviceID, appID)
	if se != nil {
		_ = se.sess.SendDebug("setBreakpoints", map[string]any{"source": source, "lines": lines})
	}
	out := make([]DebugBreakpoint, 0, len(lines))
	for i, ln := range lines {
		out = append(out, DebugBreakpoint{ID: i + 1, Verified: ln > 0, Line: ln, Source: source})
	}
	return out
}

// DebugCommand forwards a step/continue/pause command.
func (s *Supervisor) DebugCommand(deviceID, appID, command string) error {
	se := s.get(deviceID, appID)
	if se == nil {
		return emulator.ErrAppNotFound
	}
	return se.sess.SendDebug(command, nil)
}

// DebugEvaluate evaluates a simple expression against session data.
func (s *Supervisor) DebugEvaluate(deviceID, appID, expr string) (string, error) {
	se := s.get(deviceID, appID)
	if se == nil {
		return "", emulator.ErrAppNotFound
	}
	expr = strings.TrimSpace(expr)
	frames, patches := se.sess.Stats()
	switch expr {
	case "state":
		return se.sess.State(), nil
	case "session":
		return se.sess.ID, nil
	case "frames":
		return fmt.Sprintf("%d", frames), nil
	case "patches":
		return fmt.Sprintf("%d", patches), nil
	case "pid":
		return fmt.Sprintf("%d", se.sess.Pid()), nil
	}
	// Try the saved state blob as JSON.
	if saved := se.sess.SavedState(); len(saved) > 0 {
		var m map[string]any
		if json.Unmarshal(saved, &m) == nil {
			if v, ok := m[expr]; ok {
				return fmt.Sprintf("%v", v), nil
			}
		}
	}
	return "(nil)", nil
}

// DebugTree returns the last UI tree, used by the inspector.
func (s *Supervisor) DebugTree(deviceID, appID string) json.RawMessage {
	if se := s.get(deviceID, appID); se != nil {
		return se.sess.LastTree()
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var _ = sort.Ints
