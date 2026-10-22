package emulator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppLauncher is implemented by the runtime supervisor. It decouples the
// device manager from the worker-process machinery and prevents import cycles.
type AppLauncher interface {
	Launch(dev *Device, installed InstalledApp) (sessionID string, pid int, err error)
	Stop(dev *Device, appID string) error
	Pause(dev *Device, appID string) error
	Resume(dev *Device, appID string) error
}

// Manager owns all virtual devices and their persisted state.
type Manager struct {
	mu       sync.RWMutex
	dataDir  string
	devices  map[string]*Device
	profiles map[string]DeviceProfile
	logHub   *LogHub
	launcher AppLauncher
	guard    PathGuard
}

// PathGuard is the minimal interface used to validate host paths.
type PathGuard interface {
	Check(path string) (string, error)
	Within(path string) bool
}

// Device is a single virtual device instance.
type Device struct {
	mu sync.Mutex

	ID         string                  `json:"id"`
	Serial     string                  `json:"serial"`
	Name       string                  `json:"name"`
	ProfileID  string                  `json:"profile_id"`
	State      DeviceState             `json:"state,omitempty"`
	CreatedAt  int64                   `json:"created_at"`
	LastBootAt int64                   `json:"last_boot_at"`
	Installed  map[string]InstalledApp `json:"installed,omitempty"`
	Running    map[string]AppInstance  `json:"running,omitempty"`
	Settings   DeviceSettings          `json:"settings"`
	StorageDir string                  `json:"storage_dir,omitempty"`
	cancelBoot context.CancelFunc
}

// NewManager opens (or creates) a device store under dataDir.
func NewManager(dataDir string) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	m := &Manager{
		dataDir:  dataDir,
		devices:  make(map[string]*Device),
		profiles: LoadProfiles(filepath.Join(dataDir, "profiles")),
		logHub:   NewLogHub(10000),
	}
	// Merge profiles shipped with the repository (devices/profiles/*.json).
	for id, p := range LoadProfiles(filepath.Join("devices", "profiles")) {
		if _, exists := m.profiles[id]; !exists {
			m.profiles[id] = p
		}
	}
	if err := m.loadDevices(); err != nil {
		return nil, err
	}
	return m, nil
}

// SetLauncher installs the app launcher.
func (m *Manager) SetLauncher(l AppLauncher) { m.launcher = l }

// SetGuard installs a path guard for filesystem operations.
func (m *Manager) SetGuard(g PathGuard) { m.guard = g }

// DataDir returns the manager root.
func (m *Manager) DataDir() string { return m.dataDir }

// Logs returns the shared log hub.
func (m *Manager) Logs() *LogHub { return m.logHub }

// Profiles returns a copy of the known device profiles.
func (m *Manager) Profiles() map[string]DeviceProfile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]DeviceProfile, len(m.profiles))
	for k, v := range m.profiles {
		out[k] = v
	}
	return out
}

// Profile returns a single profile.
func (m *Manager) Profile(id string) (DeviceProfile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.profiles[id]
	return p, ok
}

// CreateDevice provisions a new virtual device from a profile.
func (m *Manager) CreateDevice(profileID, name string) (*DeviceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	profile, ok := m.profiles[profileID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProfile, profileID)
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = fmt.Sprintf("%s #%s", profile.DisplayName, id[:4])
	}

	storageDir := filepath.Join(m.dataDir, "devices", id)
	for _, sub := range []string{"system", "data/apps", "cache", "tmp", "snapshots"} {
		if err := os.MkdirAll(filepath.Join(storageDir, filepath.FromSlash(sub)), 0o755); err != nil {
			return nil, err
		}
	}

	dev := &Device{
		ID:         id,
		Serial:     serial,
		Name:       name,
		ProfileID:  profileID,
		State:      StateOff,
		CreatedAt:  nowMillis(),
		Installed:  map[string]InstalledApp{},
		Running:    map[string]AppInstance{},
		StorageDir: storageDir,
		Settings: DeviceSettings{
			Theme:          "dark",
			Locale:         "bn-BD",
			AnimationScale: 1.0,
			FontScale:      1.0,
			WifiEnabled:    true,
			BatteryLevel:   clamp(profile.Battery.MaxCharge, 0, 100),
			NetworkMode:    "wifi",
			Volume:         75,
			Brightness:     85,
		},
	}
	m.devices[id] = dev
	if err := dev.save(); err != nil {
		return nil, err
	}
	m.logHub.Emit(LogEntry{DeviceID: id, Level: "info", Tag: "device", Message: "device created: " + name})
	info := dev.toInfo(profile)
	return &info, nil
}

// ListDevices returns all devices.
func (m *Manager) ListDevices() []DeviceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]DeviceInfo, 0, len(m.devices))
	for _, dev := range m.devices {
		out = append(out, dev.toInfo(m.profiles[dev.ProfileID]))
	}
	return out
}

// GetDevice returns a device by id.
func (m *Manager) GetDevice(id string) (*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dev, ok := m.devices[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return dev, nil
}

// DeviceInfo returns the JSON snapshot of a device.
func (m *Manager) DeviceInfo(id string) (DeviceInfo, error) {
	m.mu.RLock()
	dev, ok := m.devices[id]
	profile := DeviceProfile{}
	if ok {
		profile = m.profiles[dev.ProfileID]
	}
	m.mu.RUnlock()
	if !ok {
		return DeviceInfo{}, ErrDeviceNotFound
	}
	return dev.toInfo(profile), nil
}

// StartDevice boots a device.
func (m *Manager) StartDevice(ctx context.Context, id string) error {
	dev, err := m.GetDevice(id)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	if dev.State == StateRunning || dev.State == StateBooting {
		dev.mu.Unlock()
		return nil
	}
	// Boot must not be tied to the caller's context (e.g. an HTTP request),
	// which is cancelled as soon as the handler returns.
	_ = ctx
	bootCtx, cancel := context.WithCancel(context.Background())
	dev.cancelBoot = cancel
	dev.State = StateBooting
	dev.LastBootAt = nowMillis()
	dev.mu.Unlock()

	go m.bootDevice(bootCtx, dev)
	return nil
}

func (m *Manager) bootDevice(ctx context.Context, dev *Device) {
	defer func() {
		dev.mu.Lock()
		dev.cancelBoot = nil
		dev.mu.Unlock()
	}()
	m.logHub.Emit(LogEntry{DeviceID: dev.ID, Level: "info", Tag: "boot", Message: "NilOS runtime booting..."})
	select {
	case <-time.After(250 * time.Millisecond):
	case <-ctx.Done():
		dev.mu.Lock()
		dev.State = StateOff
		dev.mu.Unlock()
		return
	}
	dev.mu.Lock()
	dev.State = StateRunning
	dev.mu.Unlock()
	_ = dev.save()
	m.logHub.Emit(LogEntry{DeviceID: dev.ID, Level: "info", Tag: "boot", Message: "NilOS runtime ready."})
}

// StopDevice stops a device and its apps.
func (m *Manager) StopDevice(id string) error {
	dev, err := m.GetDevice(id)
	if err != nil {
		return err
	}
	if m.launcher != nil {
		for appID := range dev.snapshotRunning() {
			_ = m.launcher.Stop(dev, appID)
		}
	}

	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.cancelBoot != nil {
		dev.cancelBoot()
		dev.cancelBoot = nil
	}
	for appID, inst := range dev.Running {
		inst.State = LifecycleDestroyed
		dev.Running[appID] = inst
	}
	dev.Running = map[string]AppInstance{}
	dev.State = StateOff
	return dev.save()
}

// DeleteDevice removes a device and its storage.
func (m *Manager) DeleteDevice(id string) error {
	m.mu.Lock()
	dev, ok := m.devices[id]
	if !ok {
		m.mu.Unlock()
		return ErrDeviceNotFound
	}
	delete(m.devices, id)
	m.mu.Unlock()

	dev.mu.Lock()
	if dev.cancelBoot != nil {
		dev.cancelBoot()
	}
	dev.mu.Unlock()

	if err := os.RemoveAll(dev.StorageDir); err != nil {
		return err
	}
	m.logHub.Emit(LogEntry{DeviceID: id, Level: "info", Tag: "device", Message: "device deleted"})
	return nil
}

// SetSetting mutates a device setting.
func (m *Manager) SetSetting(id string, apply func(*DeviceSettings)) error {
	dev, err := m.GetDevice(id)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	apply(&dev.Settings)
	dev.Settings.BatteryLevel = clamp(dev.Settings.BatteryLevel, 0, 100)
	err = dev.save()
	dev.mu.Unlock()
	return err
}

// UpdateAppState records an app instance state transition.
func (m *Manager) UpdateAppState(deviceID, appID string, state LifecycleState, sessionID string, pid int) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if state == LifecycleDestroyed || state == LifecycleStopped {
		delete(dev.Running, appID)
		return nil
	}
	inst := dev.Running[appID]
	if inst.ID == "" {
		inst = AppInstance{ID: appID, DeviceID: deviceID, AppID: appID, StartedAt: nowMillis()}
	}
	inst.State = state
	if sessionID != "" {
		inst.SessionID = sessionID
	}
	if pid > 0 {
		inst.PidHint = pid
	}
	dev.Running[appID] = inst
	return nil
}

func (m *Manager) loadDevices() error {
	root := filepath.Join(m.dataDir, "devices")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dev := &Device{}
		path := filepath.Join(root, e.Name(), "device.json")
		if err := loadJSON(path, dev); err != nil {
			continue
		}
		dev.Installed = map[string]InstalledApp{}
		dev.Running = map[string]AppInstance{}
		dev.StorageDir = filepath.Join(root, e.Name())
		_ = dev.loadInstalled()
		dev.State = StateOff
		dev.cancelBoot = nil
		m.devices[dev.ID] = dev
	}
	return nil
}

// SettingsSnapshot returns a copy of the device settings and profile id.
func (d *Device) SettingsSnapshot() (DeviceSettings, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Settings, d.ProfileID
}

func (d *Device) toInfo(profile DeviceProfile) DeviceInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DeviceInfo{
		ID:            d.ID,
		Serial:        d.Serial,
		Name:          d.Name,
		ProfileID:     d.ProfileID,
		State:         d.State,
		CreatedAt:     d.CreatedAt,
		LastBootAt:    d.LastBootAt,
		InstalledApps: copyMap(d.Installed),
		Settings:      d.Settings,
		RunningApps:   copyMap(d.Running),
	}
}

func (d *Device) save() error {
	return saveJSON(filepath.Join(d.StorageDir, "device.json"), d.exportPersistable())
}

func (d *Device) exportPersistable() map[string]any {
	return map[string]any{
		"id":           d.ID,
		"serial":       d.Serial,
		"name":         d.Name,
		"profile_id":   d.ProfileID,
		"created_at":   d.CreatedAt,
		"last_boot_at": d.LastBootAt,
		"settings":     d.Settings,
	}
}

func (d *Device) loadInstalled() error {
	path := filepath.Join(d.StorageDir, "installed.json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return loadJSON(path, &d.Installed)
}

func (d *Device) saveInstalled() error {
	return saveJSON(filepath.Join(d.StorageDir, "installed.json"), d.Installed)
}

// snapshotRunning copies the running-app map under the device lock.
func (d *Device) snapshotRunning() map[string]AppInstance {
	d.mu.Lock()
	defer d.mu.Unlock()
	return copyMap(d.Running)
}

// AppDataDir returns the sandbox directory for an installed app.
func (d *Device) AppDataDir(appID string) string {
	return filepath.Join(d.StorageDir, "data", "apps", sanitizeID(appID))
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomSerial() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "NIL-" + hex.EncodeToString(b), nil
}

func copyMap[V any](src map[string]V) map[string]V {
	dst := make(map[string]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// sanitizeID keeps only filename-safe characters from an app id.
func sanitizeID(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "app"
	}
	return string(out)
}
