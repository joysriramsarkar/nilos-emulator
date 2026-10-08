package sandbox

// Limits describes resource constraints applied to an app worker process.
type Limits struct {
	// MemoryMB caps the resident/committed memory of the worker. 0 = none.
	MemoryMB int
	// CPUPct caps CPU usage as a percentage of one core. 0 = none.
	CPUPct int
}

// Release undoes whatever Attach did; it is always safe to call.
type Release func()

// Attach applies the limits to an already-started process identified by pid.
// It is best-effort: on platforms or kernels where the mechanism is
// unavailable it returns (nil, nil) so callers continue without limits.
func Attach(pid int, l Limits) (Release, error) {
	if pid <= 0 {
		return nil, nil
	}
	return attach(pid, l)
}

// ConfigureCmd can install any platform-specific pre-exec configuration.
// Currently a no-op placeholder so callers have a stable API.
func ConfigureCmd(cfg any, l Limits) error { return nil }
