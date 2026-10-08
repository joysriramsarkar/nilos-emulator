package emulator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
)

// InstallBundle verifies and installs a .nilax bundle on a device.
func (m *Manager) InstallBundle(deviceID, bundlePath string, opts packages.Options) (*InstalledApp, error) {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}

	appsRoot := filepath.Join(dev.StorageDir, "data", "apps")
	if err := os.MkdirAll(appsRoot, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(appsRoot, ".install-*")
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tmp)
		}
	}()

	bundle, err := packages.LoadAndVerify(bundlePath, tmp, opts)
	if err != nil {
		return nil, err
	}
	man := bundle.Manifest
	finalDir := dev.AppDataDir(man.ID)
	if err := os.RemoveAll(finalDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, finalDir); err != nil {
		return nil, err
	}
	success = true

	perms := map[string]bool{}
	for _, p := range man.Permissions {
		perms[strings.ToLower(strings.TrimSpace(p))] = true
	}

	app := InstalledApp{
		ID:             man.ID,
		Name:           man.Name,
		Version:        man.Version,
		Entry:          man.Entry,
		Icon:           man.Icon,
		InstalledAt:    nowMillis(),
		Enabled:        true,
		Permissions:    perms,
		InstallPath:    finalDir,
		SignatureKeyID: "",
		Signed:         bundle.Signature != nil,
		Devices:        man.Devices,
	}
	if man.Limits != nil {
		app.Limits = AppLimits{MemoryMB: man.Limits.MemoryMB, CPUPct: man.Limits.CPUPct}
	}
	if bundle.Signature != nil {
		app.SignatureKeyID = bundle.Signature.KeyID
	}

	dev.mu.Lock()
	dev.Installed[man.ID] = app
	err = dev.saveInstalled()
	dev.mu.Unlock()
	if err != nil {
		return nil, err
	}

	m.logHub.Emit(LogEntry{DeviceID: deviceID, AppID: man.ID, Level: "info", Tag: "pkg",
		Message: fmt.Sprintf("installed %s v%s", man.Name, man.Version)})
	return &app, nil
}

// ListInstalled returns installed apps sorted by id.
func (m *Manager) ListInstalled(deviceID string) ([]InstalledApp, error) {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	dev.mu.Lock()
	out := make([]InstalledApp, 0, len(dev.Installed))
	for _, a := range dev.Installed {
		out = append(out, a)
	}
	dev.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// GetInstalled returns a single installed app.
func (m *Manager) GetInstalled(deviceID, appID string) (InstalledApp, error) {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return InstalledApp{}, err
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	app, ok := dev.Installed[appID]
	if !ok {
		return InstalledApp{}, ErrAppNotFound
	}
	return app, nil
}

// Uninstall removes an app and its data.
func (m *Manager) Uninstall(deviceID, appID string) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	if m.launcher != nil {
		_ = m.launcher.Stop(dev, appID)
	}
	dev.mu.Lock()
	app, ok := dev.Installed[appID]
	if !ok {
		dev.mu.Unlock()
		return ErrAppNotFound
	}
	delete(dev.Installed, appID)
	delete(dev.Running, appID)
	err = dev.saveInstalled()
	dev.mu.Unlock()
	if err != nil {
		return err
	}
	if app.InstallPath != "" {
		_ = os.RemoveAll(app.InstallPath)
	}
	m.logHub.Emit(LogEntry{DeviceID: deviceID, AppID: appID, Level: "info", Tag: "pkg", Message: "uninstalled"})
	return nil
}

// ClearAppData wipes an app's private storage but keeps it installed.
func (m *Manager) ClearAppData(deviceID, appID string) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	_, ok := dev.Installed[appID]
	dev.mu.Unlock()
	if !ok {
		return ErrAppNotFound
	}
	dataDir := filepath.Join(dev.AppDataDir(appID), "data")
	if err := os.RemoveAll(dataDir); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	m.logHub.Emit(LogEntry{DeviceID: deviceID, AppID: appID, Level: "info", Tag: "pkg", Message: "app data cleared"})
	return nil
}

// AppEntryPath returns the absolute path to the app entry file.
func (dev *Device) AppEntryPath(app InstalledApp) string {
	return filepath.Join(app.InstallPath, filepath.FromSlash(app.Entry))
}
