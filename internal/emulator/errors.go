package emulator

import "errors"

var (
	// ErrDeviceNotFound is returned when a device id does not exist.
	ErrDeviceNotFound = errors.New("device not found")
	// ErrAppNotFound is returned when an app id is not installed.
	ErrAppNotFound = errors.New("app not found")
	// ErrDeviceNotReady is returned when an operation needs a running device.
	ErrDeviceNotReady = errors.New("device is not running")
	// ErrInvalidProfile is returned for an unknown device profile.
	ErrInvalidProfile = errors.New("invalid device profile")
	// ErrPermissionDenied is returned when a capability is not granted.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrAlreadyRunning is returned when starting an already-running app.
	ErrAlreadyRunning = errors.New("app already running")
)
