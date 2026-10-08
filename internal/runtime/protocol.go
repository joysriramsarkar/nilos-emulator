// Package runtime implements the NilHost Protocol: the versioned
// newline-delimited JSON contract between the emulator host and a NilLang
// runtime worker ("VM"). See docs/protocol.md.
package runtime

import (
	"bytes"
	"encoding/json"
)

// ProtoVersion is the NilHost protocol version implemented here.
const ProtoVersion = 1

// Insets mirrors the device safe-area insets sent in Init.
type Insets struct {
	Top    int `json:"t"`
	Bottom int `json:"b"`
	Left   int `json:"l"`
	Right  int `json:"r"`
}

// DeviceInit describes the virtual device to the VM.
type DeviceInit struct {
	Profile     string   `json:"profile"`
	W           int      `json:"w"`
	H           int      `json:"h"`
	DPR         float64  `json:"dpr"`
	Safe        Insets   `json:"safe"`
	RefreshHz   int      `json:"refresh_hz"`
	Orientation string   `json:"orientation"`
	Inputs      []string `json:"inputs,omitempty"`
}

// Hello is the first message the VM sends.
type Hello struct {
	Type    string   `json:"t"`
	Proto   int      `json:"proto"`
	Runtime string   `json:"runtime"`
	App     string   `json:"app"`
	Caps    []string `json:"caps,omitempty"`
}

// Init is the host's reply to Hello.
type Init struct {
	Type        string         `json:"t"`
	Proto       int            `json:"proto"`
	Device      DeviceInit     `json:"device"`
	Locale      string         `json:"locale"`
	Theme       string         `json:"theme"`
	FontScale   float64        `json:"font_scale,omitempty"`
	HAL         map[string]any `json:"hal,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
	AppData     string         `json:"app_data,omitempty"`
	SavedState  json.RawMessage `json:"saved_state,omitempty"`
	Seed        int64          `json:"seed,omitempty"`
	Animations  *bool          `json:"animations,omitempty"`
}

// InputMsg is a touch/key/text event from host to VM.
type InputMsg struct {
	Type   string  `json:"t"`
	Kind   string  `json:"kind"`            // touch | key | text | scroll | back | home
	Phase  string  `json:"phase,omitempty"` // down | move | up | cancel
	ID     int     `json:"id,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Ts     int64   `json:"ts,omitempty"`
	Key    string  `json:"key,omitempty"`
	Code   string  `json:"code,omitempty"`
	Text   string  `json:"text,omitempty"`
	DeltaX float64 `json:"dx,omitempty"`
	DeltaY float64 `json:"dy,omitempty"`
}

// LifecycleMsg drives app lifecycle transitions.
type LifecycleMsg struct {
	Type  string `json:"t"`
	State string `json:"state"` // create|start|resume|pause|stop|destroy
}

// ConfigMsg reports configuration changes.
type ConfigMsg struct {
	Type        string  `json:"t"`
	Theme       string  `json:"theme,omitempty"`
	Locale      string  `json:"locale,omitempty"`
	FontScale   float64 `json:"font_scale,omitempty"`
	Orientation string  `json:"orientation,omitempty"`
	W           int     `json:"w,omitempty"`
	H           int     `json:"h,omitempty"`
}

// VsyncMsg is a frame credit from the host.
type VsyncMsg struct {
	Type string `json:"t"`
	Ts   int64  `json:"ts"`
}

// ImeMsg drives the input method editor.
type ImeMsg struct {
	Type string `json:"t"`
	Op   string `json:"op"` // show|hide|commit|compose|delete
	Text string `json:"text,omitempty"`
	N    int    `json:"n,omitempty"`
}

// ReloadMsg asks the VM to hot-restart or hot-reload.
type ReloadMsg struct {
	Type string `json:"t"`
	Mode string `json:"mode"` // restart|reload
}

// DebugMsg carries a debugger command.
type DebugMsg struct {
	Type string         `json:"t"`
	Cmd  string         `json:"cmd"`
	Args map[string]any `json:"args,omitempty"`
}

// FrameMsg carries a full UI tree.
type FrameMsg struct {
	Type string          `json:"t"`
	Seq  uint64          `json:"seq,omitempty"`
	Tree json.RawMessage `json:"tree"`
}

// PatchMsg carries a list of tree diff operations.
type PatchMsg struct {
	Type string          `json:"t"`
	Seq  uint64          `json:"seq,omitempty"`
	Ops  json.RawMessage `json:"ops"`
}

// HalCallMsg is a device request from VM to host.
type HalCallMsg struct {
	Type string          `json:"t"`
	ID   uint64          `json:"id"`
	Dev  string          `json:"dev"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
}

// HalResultMsg is the host's reply to a HalCallMsg.
type HalResultMsg struct {
	Type  string          `json:"t"`
	ID    uint64          `json:"id"`
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// BusMsg is a NilBus message.
type BusMsg struct {
	Type    string          `json:"t"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// LogMsg is a log line.
type LogMsg struct {
	Type  string `json:"t"`
	Level string `json:"level"`
	Tag   string `json:"tag,omitempty"`
	Msg   string `json:"msg"`
	Ts    int64  `json:"ts,omitempty"`
}

// TraceMsg carries DevTools timing data.
type TraceMsg struct {
	Type   string  `json:"t"`
	Frames uint64  `json:"frames,omitempty"`
	Ms     float64 `json:"ms,omitempty"`
	Phase  string  `json:"phase,omitempty"`
	HeapMB float64 `json:"heap_mb,omitempty"`
}

// StateMsg carries a saved-state blob for restore.
type StateMsg struct {
	Type string          `json:"t"`
	Blob json.RawMessage `json:"blob"`
}

// ExitMsg reports worker exit.
type ExitMsg struct {
	Type   string `json:"t"`
	Code   int    `json:"code"`
	Reason string `json:"reason,omitempty"`
}

// ErrorMsg reports a runtime error.
type ErrorMsg struct {
	Type  string   `json:"t"`
	Code  string   `json:"code"`
	Msg   string   `json:"msg"`
	Stack []string `json:"stack,omitempty"`
}

// Envelope is used to peek at the message type.
type Envelope struct {
	Type string `json:"t"`
}

// encodeLine marshals v and appends a newline (the wire framing).
func encodeLine(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// validLine reports whether a raw line looks like a JSON object.
func validLine(line []byte) bool {
	trimmed := bytes.TrimSpace(line)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
