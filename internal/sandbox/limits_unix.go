//go:build !windows

package sandbox

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// attach uses cgroup v2 if it is writable, otherwise it is a no-op. The
// cgroup path is created under the current cgroup and removed on release.
func attach(pid int, l Limits) (Release, error) {
	if l.MemoryMB <= 0 {
		return nil, nil
	}
	current, err := currentCgroup()
	if err != nil {
		return nil, nil
	}
	group := filepath.Join(current, "nilemu")
	if err := os.MkdirAll(group, 0o755); err != nil {
		return nil, nil
	}
	if err := os.WriteFile(filepath.Join(group, "memory.max"), []byte(strconv.Itoa(l.MemoryMB*1024*1024)), 0o644); err != nil {
		return nil, nil
	}
	if err := os.WriteFile(filepath.Join(group, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return nil, nil
	}
	return func() { _ = os.Remove(group) }, nil
}

func currentCgroup() (string, error) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" {
			return filepath.Join("/sys/fs/cgroup", parts[2]), nil
		}
	}
	return "", os.ErrNotExist
}
