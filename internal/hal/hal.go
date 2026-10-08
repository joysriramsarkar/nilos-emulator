// Package hal implements the virtual hardware layer (NilHAL). Each device
// class is addressed by name and operation over the NilHost Protocol's
// hal.call/hal.result messages. Fault injection lets developers exercise
// error paths deterministically.
package hal

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/vfs"
)

// FaultRule makes the next N calls to a device operation fail.
type FaultRule struct {
	Dev   string `json:"dev"`
	Op    string `json:"op"`
	Code  string `json:"code"`
	Times int    `json:"times"`
}

// Config configures a Dispatcher for one device.
type Config struct {
	DeviceID  string
	VFS       *vfs.FS
	Permit    func(device, op string) error
	OnLog     func(tag, msg string)
	WidthPx   int
	HeightPx  int
	DPR       float64
	Locale    string
	ClockISO  string
	Freeze    bool
}

// Dispatcher routes hal.call operations to virtual devices.
type Dispatcher struct {
	mu sync.Mutex

	cfg Config

	battery  int
	charging bool
	tempC    float64

	networkMode string
	online      bool
	latencyMs   int

	clock   time.Time
	frozen  bool
	tz      string

	clipboard     string
	notifications []Notification
	sensorAccel   [3]float64
	sensorGyro    [3]float64
	gpsLat        float64
	gpsLng        float64

	lastReceipt []byte
	scanBuffer  string

	faults []FaultRule

	volume     int
	brightness int
}

// Notification is a posted system notification.
type Notification struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	AppID   string `json:"app_id,omitempty"`
	Created int64  `json:"created"`
}

// NewDispatcher builds a dispatcher with default virtual hardware state.
func NewDispatcher(cfg Config) *Dispatcher {
	d := &Dispatcher{
		cfg:         cfg,
		battery:     100,
		charging:    false,
		tempC:       32.0,
		networkMode: "wifi",
		online:      true,
		latencyMs:   20,
		clock:       time.Now(),
		tz:          "Asia/Dhaka",
		volume:      75,
		brightness:  85,
	}
	if cfg.ClockISO != "" {
		if t, err := time.Parse(time.RFC3339, cfg.ClockISO); err == nil {
			d.clock = t
		}
	}
	d.frozen = cfg.Freeze
	return d
}

// Call executes one HAL operation.
func (d *Dispatcher) Call(dev, op string, args json.RawMessage) (any, error) {
	if d.cfg.Permit != nil {
		if err := d.cfg.Permit(dev, op); err != nil {
			return nil, err
		}
	}
	if code, faulted := d.popFault(dev, op); faulted {
		return nil, &FaultError{Dev: dev, Op: op, Code: code}
	}

	switch dev {
	case "battery", "power":
		return d.batteryCall(op, args)
	case "network", "net":
		return d.networkCall(op, args)
	case "clock", "time":
		return d.clockCall(op, args)
	case "storage", "filesystem", "fs":
		return d.storageCall(op, args)
	case "printer":
		return d.printerCall(op, args)
	case "scanner":
		return d.scannerCall(op, args)
	case "clipboard":
		return d.clipboardCall(op, args)
	case "notifications":
		return d.notificationCall(op, args)
	case "sensors":
		return d.sensorCall(op, args)
	case "display":
		return d.displayCall(op, args)
	case "audio", "haptics":
		return d.audioCall(dev, op, args)
	case "device":
		return d.deviceCall(op, args)
	default:
		return nil, fmt.Errorf("unknown HAL device: %s", dev)
	}
}

// FaultError reports an injected failure.
type FaultError struct {
	Dev  string
	Op   string
	Code string
}

func (e *FaultError) Error() string {
	return fmt.Sprintf("hal: injected fault %s on %s.%s", e.Code, e.Dev, e.Op)
}

// InjectFault schedules a failure for a device operation.
func (d *Dispatcher) InjectFault(rule FaultRule) {
	if rule.Times <= 0 {
		rule.Times = 1
	}
	d.mu.Lock()
	d.faults = append(d.faults, rule)
	d.mu.Unlock()
}

// ClearFaults removes all injected faults.
func (d *Dispatcher) ClearFaults() {
	d.mu.Lock()
	d.faults = nil
	d.mu.Unlock()
}

func (d *Dispatcher) popFault(dev, op string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, f := range d.faults {
		if f.Dev == dev && (f.Op == op || f.Op == "*") && f.Times > 0 {
			d.faults[i].Times--
			return f.Code, true
		}
	}
	return "", false
}

// ─── device implementations ─────────────────────────────────────────────

func (d *Dispatcher) batteryCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "level", "get":
		return map[string]any{"level": d.battery, "charging": d.charging, "temperature_c": d.tempC}, nil
	case "set":
		var a struct {
			Level    *int  `json:"level"`
			Charging *bool `json:"charging"`
		}
		_ = json.Unmarshal(args, &a)
		if a.Level != nil {
			d.battery = clampInt(*a.Level, 0, 100)
		}
		if a.Charging != nil {
			d.charging = *a.Charging
		}
		return map[string]any{"level": d.battery, "charging": d.charging}, nil
	}
	return nil, fmt.Errorf("battery: unknown op %s", op)
}

func (d *Dispatcher) networkCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "status", "get":
		return map[string]any{
			"mode": d.networkMode, "online": d.online,
			"latency_ms": d.latencyMs, "down_kbps": d.downKbps(),
		}, nil
	case "set":
		var a struct {
			Mode    string `json:"mode"`
			Online  *bool  `json:"online"`
			Latency int    `json:"latency_ms"`
		}
		_ = json.Unmarshal(args, &a)
		if a.Mode != "" {
			d.networkMode = a.Mode
			d.latencyMs = defaultLatency(a.Mode)
			d.online = a.Mode != "offline"
		}
		if a.Online != nil {
			d.online = *a.Online
		}
		if a.Latency > 0 {
			d.latencyMs = a.Latency
		}
		return map[string]any{"mode": d.networkMode, "online": d.online, "latency_ms": d.latencyMs}, nil
	}
	return nil, fmt.Errorf("network: unknown op %s", op)
}

func (d *Dispatcher) downKbps() int {
	switch d.networkMode {
	case "5g":
		return 1200000
	case "4g":
		return 100000
	case "3g":
		return 3000
	case "edge":
		return 240
	case "offline":
		return 0
	default:
		return 50000
	}
}

func (d *Dispatcher) clockCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "now", "get":
		now := time.Now()
		if d.frozen {
			now = d.clock
		}
		return map[string]any{
			"unix_ms": now.UnixMilli(),
			"iso":     now.Format(time.RFC3339),
			"tz":      d.tz,
			"frozen":  d.frozen,
		}, nil
	case "set":
		var a struct {
			UnixMs int64  `json:"unix_ms"`
			ISO    string `json:"iso"`
			TZ     string `json:"tz"`
			Frozen *bool  `json:"frozen"`
		}
		_ = json.Unmarshal(args, &a)
		if a.ISO != "" {
			if t, err := time.Parse(time.RFC3339, a.ISO); err == nil {
				d.clock = t
			}
		} else if a.UnixMs > 0 {
			d.clock = time.UnixMilli(a.UnixMs)
		}
		if a.TZ != "" {
			d.tz = a.TZ
		}
		if a.Frozen != nil {
			d.frozen = *a.Frozen
		}
		return map[string]any{"unix_ms": d.clock.UnixMilli(), "tz": d.tz, "frozen": d.frozen}, nil
	}
	return nil, fmt.Errorf("clock: unknown op %s", op)
}

func (d *Dispatcher) storageCall(op string, args json.RawMessage) (any, error) {
	if d.cfg.VFS == nil {
		return nil, fmt.Errorf("storage: no virtual filesystem")
	}
	switch op {
	case "read":
		var a struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(args, &a)
		data, err := d.cfg.VFS.ReadFile(a.Path)
		if err != nil {
			return nil, err
		}
		return map[string]any{"path": a.Path, "data": string(data), "size": len(data)}, nil
	case "write":
		var a struct {
			Path string `json:"path"`
			Data string `json:"data"`
		}
		_ = json.Unmarshal(args, &a)
		if err := d.cfg.VFS.WriteFile(a.Path, []byte(a.Data)); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "bytes": len(a.Data)}, nil
	case "list":
		var a struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(args, &a)
		entries, err := d.cfg.VFS.List(a.Path)
		if err != nil {
			return nil, err
		}
		return map[string]any{"entries": entries}, nil
	case "delete", "remove":
		var a struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(args, &a)
		if err := d.cfg.VFS.Remove(a.Path); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "exists":
		var a struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(args, &a)
		return map[string]any{"exists": d.cfg.VFS.Exists(a.Path)}, nil
	}
	return nil, fmt.Errorf("storage: unknown op %s", op)
}

func (d *Dispatcher) printerCall(op string, args json.RawMessage) (any, error) {
	switch op {
	case "print":
		var a struct {
			Lines []string `json:"lines"`
			Data  string   `json:"data"`
		}
		_ = json.Unmarshal(args, &a)
		var payload []byte
		if a.Data != "" {
			payload = []byte(a.Data)
		} else {
			payload = EncodeReceipt(a.Lines)
		}
		d.mu.Lock()
		d.lastReceipt = payload
		d.mu.Unlock()
		receipt := DecodeESCPS(payload)
		return map[string]any{"ok": true, "bytes": len(payload), "receipt": receipt}, nil
	case "list":
		return []map[string]any{{"id": "printer-0", "name": "NilThermal 58mm", "kind": "escpos"}}, nil
	case "last":
		d.mu.Lock()
		defer d.mu.Unlock()
		return map[string]any{"bytes": len(d.lastReceipt), "receipt": DecodeESCPS(d.lastReceipt)}, nil
	}
	return nil, fmt.Errorf("printer: unknown op %s", op)
}

func (d *Dispatcher) scannerCall(op string, args json.RawMessage) (any, error) {
	switch op {
	case "scan":
		var a struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(args, &a)
		format := "EAN13"
		if len(a.Value) > 0 && (a.Value[0] == '{' || len(a.Value) < 8) {
			format = "CODE128"
		}
		return map[string]any{"format": format, "value": a.Value, "ts": time.Now().UnixMilli()}, nil
	case "wedge":
		// Keyboard-wedge scanners emit fast keystrokes ending with Enter.
		var a struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(args, &a)
		d.mu.Lock()
		d.scanBuffer = a.Value
		d.mu.Unlock()
		keys := make([]string, 0, len(a.Value)+1)
		for _, r := range a.Value {
			keys = append(keys, string(r))
		}
		keys = append(keys, "Enter")
		return map[string]any{"keys": keys}, nil
	}
	return nil, fmt.Errorf("scanner: unknown op %s", op)
}

func (d *Dispatcher) clipboardCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "get":
		return map[string]any{"text": d.clipboard}, nil
	case "set":
		var a struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(args, &a)
		d.clipboard = a.Text
		return map[string]any{"ok": true}, nil
	}
	return nil, fmt.Errorf("clipboard: unknown op %s", op)
}

func (d *Dispatcher) notificationCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "post":
		var a struct {
			Title string `json:"title"`
			Body  string `json:"body"`
			AppID string `json:"app_id"`
		}
		_ = json.Unmarshal(args, &a)
		n := Notification{
			ID:      fmt.Sprintf("notif-%d", time.Now().UnixNano()),
			Title:   a.Title, Body: a.Body, AppID: a.AppID, Created: time.Now().UnixMilli(),
		}
		d.notifications = append(d.notifications, n)
		return n, nil
	case "list":
		return d.notifications, nil
	case "clear":
		d.notifications = nil
		return map[string]any{"ok": true}, nil
	case "dismiss":
		var a struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(args, &a)
		out := d.notifications[:0]
		for _, n := range d.notifications {
			if n.ID != a.ID {
				out = append(out, n)
			}
		}
		d.notifications = out
		return map[string]any{"ok": true}, nil
	}
	return nil, fmt.Errorf("notifications: unknown op %s", op)
}

func (d *Dispatcher) sensorCall(op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "accel":
		return map[string]any{"x": d.sensorAccel[0], "y": d.sensorAccel[1], "z": d.sensorAccel[2]}, nil
	case "gyro":
		return map[string]any{"x": d.sensorGyro[0], "y": d.sensorGyro[1], "z": d.sensorGyro[2]}, nil
	case "gps":
		return map[string]any{"lat": d.gpsLat, "lng": d.gpsLng, "accuracy_m": 5.0}, nil
	case "set":
		var a struct {
			Accel *[3]float64 `json:"accel"`
			Gyro  *[3]float64 `json:"gyro"`
			Lat   *float64    `json:"lat"`
			Lng   *float64    `json:"lng"`
		}
		_ = json.Unmarshal(args, &a)
		if a.Accel != nil {
			d.sensorAccel = *a.Accel
		}
		if a.Gyro != nil {
			d.sensorGyro = *a.Gyro
		}
		if a.Lat != nil {
			d.gpsLat = *a.Lat
		}
		if a.Lng != nil {
			d.gpsLng = *a.Lng
		}
		return map[string]any{"ok": true}, nil
	}
	return nil, fmt.Errorf("sensors: unknown op %s", op)
}

func (d *Dispatcher) displayCall(op string, args json.RawMessage) (any, error) {
	switch op {
	case "info", "get":
		return map[string]any{
			"width_px": d.cfg.WidthPx, "height_px": d.cfg.HeightPx,
			"dpr": d.cfg.DPR, "locale": d.cfg.Locale,
		}, nil
	}
	return nil, fmt.Errorf("display: unknown op %s", op)
}

func (d *Dispatcher) audioCall(dev, op string, args json.RawMessage) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch op {
	case "vibrate":
		return map[string]any{"ok": true, "device": dev}, nil
	case "set":
		var a struct {
			Volume     *int `json:"volume"`
			Brightness *int `json:"brightness"`
		}
		_ = json.Unmarshal(args, &a)
		if a.Volume != nil {
			d.volume = clampInt(*a.Volume, 0, 100)
		}
		if a.Brightness != nil {
			d.brightness = clampInt(*a.Brightness, 0, 100)
		}
		return map[string]any{"volume": d.volume, "brightness": d.brightness}, nil
	}
	return map[string]any{"ok": true}, nil
}

func (d *Dispatcher) deviceCall(op string, args json.RawMessage) (any, error) {
	switch op {
	case "info", "get":
		return map[string]any{
			"id": d.cfg.DeviceID, "width_px": d.cfg.WidthPx,
			"height_px": d.cfg.HeightPx, "locale": d.cfg.Locale,
		}, nil
	}
	return nil, fmt.Errorf("device: unknown op %s", op)
}

// State is a JSON-friendly snapshot for the control panel.
type State struct {
	Battery      int            `json:"battery"`
	Charging     bool           `json:"charging"`
	NetworkMode  string         `json:"network_mode"`
	Online       bool           `json:"online"`
	LatencyMs    int            `json:"latency_ms"`
	Volume       int            `json:"volume"`
	Brightness   int            `json:"brightness"`
	Notifications []Notification `json:"notifications,omitempty"`
}

// Snapshot returns the current virtual hardware state.
func (d *Dispatcher) Snapshot() State {
	d.mu.Lock()
	defer d.mu.Unlock()
	return State{
		Battery: d.battery, Charging: d.charging,
		NetworkMode: d.networkMode, Online: d.online, LatencyMs: d.latencyMs,
		Volume: d.volume, Brightness: d.brightness,
		Notifications: append([]Notification(nil), d.notifications...),
	}
}

// SetNetwork applies a named network profile.
func (d *Dispatcher) SetNetwork(mode string) (State, error) {
	_, err := d.networkCall("set", mustArgs(map[string]any{"mode": mode}))
	return d.Snapshot(), err
}

// SetBattery sets the battery level and charging flag.
func (d *Dispatcher) SetBattery(level int, charging bool) (State, error) {
	_, err := d.batteryCall("set", mustArgs(map[string]any{"level": level, "charging": charging}))
	return d.Snapshot(), err
}

// HardwareInit returns the HAL section of the Init message.
func (d *Dispatcher) HardwareInit() map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	return map[string]any{
		"battery": map[string]any{"level": d.battery, "charging": d.charging},
		"net":     map[string]any{"type": d.networkMode, "online": d.online, "rtt": d.latencyMs},
		"clock":   map[string]any{"unix_ms": d.clock.UnixMilli(), "tz": d.tz},
	}
}

func mustArgs(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func defaultLatency(mode string) int {
	switch mode {
	case "5g":
		return 20
	case "4g":
		return 80
	case "3g":
		return 300
	case "edge":
		return 800
	case "offline":
		return 0
	default:
		return 50
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
