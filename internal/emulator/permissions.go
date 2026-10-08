package emulator

import "strings"

// GrantPermission grants a capability to an installed app.
func (m *Manager) GrantPermission(deviceID, appID, perm string) error {
	return m.mutatePermission(deviceID, appID, perm, true)
}

// RevokePermission revokes a capability from an installed app.
func (m *Manager) RevokePermission(deviceID, appID, perm string) error {
	return m.mutatePermission(deviceID, appID, perm, false)
}

func (m *Manager) mutatePermission(deviceID, appID, perm string, granted bool) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	perm = strings.ToLower(strings.TrimSpace(perm))
	dev.mu.Lock()
	defer dev.mu.Unlock()
	app, ok := dev.Installed[appID]
	if !ok {
		return ErrAppNotFound
	}
	if app.Permissions == nil {
		app.Permissions = map[string]bool{}
	}
	app.Permissions[perm] = granted
	dev.Installed[appID] = app
	return dev.saveInstalled()
}

// PermissionGranted reports whether an app currently holds a capability.
// An empty appID (system/shell) is always granted.
func (m *Manager) PermissionGranted(deviceID, appID, perm string) bool {
	if appID == "" {
		return true
	}
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return false
	}
	perm = strings.ToLower(strings.TrimSpace(perm))
	dev.mu.Lock()
	defer dev.mu.Unlock()
	app, ok := dev.Installed[appID]
	if !ok {
		return false
	}
	return app.Permissions[perm]
}

// ResetPermissions restores an app's permissions to its manifest defaults.
func (m *Manager) ResetPermissions(deviceID, appID string) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	app, ok := dev.Installed[appID]
	if !ok {
		return ErrAppNotFound
	}
	// We do not persist the original manifest list, so reset clears grants
	// and re-grants storage/ui which are considered safe defaults.
	app.Permissions = map[string]bool{"filesystem": true, "storage": true, "ui": true}
	dev.Installed[appID] = app
	return dev.saveInstalled()
}
