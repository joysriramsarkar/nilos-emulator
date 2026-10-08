package emulator

import "time"

// DeviceState is the power/boot state of a virtual device.
type DeviceState string

const (
	StateOff     DeviceState = "off"
	StateBooting DeviceState = "booting"
	StateRunning DeviceState = "running"
	StatePaused  DeviceState = "paused"
	StateError   DeviceState = "error"
)

// LifecycleState is the lifecycle of a running app instance.
type LifecycleState string

const (
	LifecycleCreated   LifecycleState = "created"
	LifecycleStarted   LifecycleState = "started"
	LifecycleResumed   LifecycleState = "resumed"
	LifecyclePaused    LifecycleState = "paused"
	LifecycleStopped   LifecycleState = "stopped"
	LifecycleDestroyed LifecycleState = "destroyed"
	LifecycleCrashed   LifecycleState = "crashed"
	LifecycleInstalled LifecycleState = "installed"
)

// Insets are logical-pixel safe-area insets.
type Insets struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// ScreenSpec describes a device display.
type ScreenSpec struct {
	WidthPx        int     `json:"width_px"`
	HeightPx       int     `json:"height_px"`
	Density        float64 `json:"density"`
	RefreshHz      int     `json:"refresh_hz"`
	SafeArea       Insets  `json:"safe_area"`
	DynamicIsland  bool    `json:"dynamic_island"`
	Notch          bool    `json:"notch"`
	RoundedCorners int     `json:"rounded_corners"`
	Orientation    string  `json:"orientation"`
}

// InputSpec lists available input methods.
type InputSpec struct {
	Touch          bool `json:"touch"`
	Keyboard       bool `json:"keyboard"`
	Mouse          bool `json:"mouse"`
	BarcodeScanner bool `json:"barcode_scanner"`
	ThermalPrinter bool `json:"thermal_printer"`
	VirtualKeys    bool `json:"virtual_keys"`
}

// SensorSpec lists available sensors.
type SensorSpec struct {
	Accelerometer bool `json:"accelerometer"`
	Gyroscope     bool `json:"gyroscope"`
	Proximity     bool `json:"proximity"`
	AmbientLight  bool `json:"ambient_light"`
	GPS           bool `json:"gps"`
	Fingerprint   bool `json:"fingerprint"`
}

// BatterySpec describes the hardware battery.
type BatterySpec struct {
	CapacityMah int `json:"capacity_mah"`
	MaxCharge   int `json:"max_charge"`
	Health      int `json:"health"`
}

// DeviceProfile is a hardware template (AVD analogue).
type DeviceProfile struct {
	ID          string      `json:"id"`
	DisplayName string      `json:"display_name"`
	Category    string      `json:"category"`
	Screen      ScreenSpec  `json:"screen"`
	Input       InputSpec   `json:"input"`
	Sensors     SensorSpec  `json:"sensors"`
	Battery     BatterySpec `json:"battery"`
	Features    []string    `json:"features"`
}

// DeviceSettings are user-controllable device state values.
type DeviceSettings struct {
	Theme          string  `json:"theme"`
	Locale         string  `json:"locale"`
	DeveloperMode  bool    `json:"developer_mode"`
	AnimationScale float64 `json:"animation_scale"`
	FontScale      float64 `json:"font_scale"`
	WifiEnabled    bool    `json:"wifi_enabled"`
	AirplaneMode   bool    `json:"airplane_mode"`
	BatteryLevel   int     `json:"battery_level"`
	Charging       bool    `json:"charging"`
	NetworkMode    string  `json:"network_mode"`
	Volume         int     `json:"volume"`
	Brightness     int     `json:"brightness"`
}

// AppLimits caps an app worker's resource usage.
type AppLimits struct {
	MemoryMB int `json:"memory_mb,omitempty"`
	CPUPct   int `json:"cpu_pct,omitempty"`
}

// InstalledApp records an installed .nilax application.
type InstalledApp struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Entry          string          `json:"entry"`
	Icon           string          `json:"icon"`
	InstalledAt    int64           `json:"installed_at"`
	Enabled        bool            `json:"enabled"`
	Permissions    map[string]bool `json:"permissions"`
	InstallPath    string          `json:"install_path"`
	SignatureKeyID string          `json:"signature_key_id"`
	Signed         bool            `json:"signed"`
	Limits         AppLimits       `json:"limits,omitempty"`
	Devices        []string        `json:"devices,omitempty"`
}

// AppInstance is a running instance of an installed app.
type AppInstance struct {
	ID        string         `json:"id"`
	DeviceID  string         `json:"device_id"`
	AppID     string         `json:"app_id"`
	SessionID string         `json:"session_id"`
	State     LifecycleState `json:"state"`
	StartedAt int64          `json:"started_at"`
	WindowID  string         `json:"window_id,omitempty"`
	PidHint   int            `json:"pid_hint,omitempty"`
}

// DeviceInfo is the JSON-facing snapshot of a device.
type DeviceInfo struct {
	ID            string                 `json:"id"`
	Serial        string                 `json:"serial"`
	Name          string                 `json:"name"`
	ProfileID     string                 `json:"profile_id"`
	State         DeviceState            `json:"state"`
	CreatedAt     int64                  `json:"created_at"`
	LastBootAt    int64                  `json:"last_boot_at"`
	InstalledApps map[string]InstalledApp `json:"installed_apps"`
	Settings      DeviceSettings         `json:"settings"`
	RunningApps   map[string]AppInstance `json:"running_apps"`
}

// nowMillis returns the current Unix time in milliseconds.
func nowMillis() int64 { return time.Now().UnixMilli() }
