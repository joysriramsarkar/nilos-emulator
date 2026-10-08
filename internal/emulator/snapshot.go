package emulator

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SnapshotInfo describes a saved device snapshot.
type SnapshotInfo struct {
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	SizeBytes int64  `json:"size_bytes"`
}

// SaveSnapshot archives the device state (config + app data) into
// <device>/snapshots/<name>.zip.
func (m *Manager) SaveSnapshot(deviceID, name string) (*SnapshotInfo, error) {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	if name == "" || strings.ContainsAny(name, `/\:`) {
		return nil, ErrInvalidProfile
	}
	if err := dev.save(); err != nil {
		return nil, err
	}
	snapDir := filepath.Join(dev.StorageDir, "snapshots")
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return nil, err
	}
	out := filepath.Join(snapDir, name+".zip")

	f, err := os.Create(out)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(f)

	walkErr := filepath.Walk(dev.StorageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dev.StorageDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(rel, "snapshots/") {
			return nil
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(w, src)
		return err
	})
	if walkErr != nil {
		zw.Close()
		f.Close()
		os.Remove(out)
		return nil, walkErr
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	fi, err := os.Stat(out)
	if err != nil {
		return nil, err
	}
	m.logHub.Emit(LogEntry{DeviceID: deviceID, Level: "info", Tag: "snapshot", Message: "saved snapshot " + name})
	return &SnapshotInfo{Name: name, CreatedAt: nowMillis(), SizeBytes: fi.Size()}, nil
}

// ListSnapshots lists a device's snapshots.
func (m *Manager) ListSnapshots(deviceID string) ([]SnapshotInfo, error) {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(dev.StorageDir, "snapshots"))
	if err != nil {
		return []SnapshotInfo{}, nil
	}
	var out []SnapshotInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".zip" {
			continue
		}
		info, _ := e.Info()
		var size int64
		var created int64
		if info != nil {
			size = info.Size()
			created = info.ModTime().UnixMilli()
		}
		out = append(out, SnapshotInfo{
			Name:      strings.TrimSuffix(e.Name(), ".zip"),
			CreatedAt: created,
			SizeBytes: size,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// RestoreSnapshot replaces device state and app data from a snapshot.
func (m *Manager) RestoreSnapshot(deviceID, name string) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	zipPath := filepath.Join(dev.StorageDir, "snapshots", name+".zip")
	if _, err := os.Stat(zipPath); err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	// Wipe current app/user data (keep snapshots).
	for _, sub := range []string{"data", "cache", "tmp", "system"} {
		_ = os.RemoveAll(filepath.Join(dev.StorageDir, sub))
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "snapshots/") {
			continue
		}
		target, err := safeJoin(dev.StorageDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	// Reload persisted state into memory.
	dev.mu.Lock()
	dev.Installed = map[string]InstalledApp{}
	_ = dev.loadInstalled()
	dev.mu.Unlock()
	m.logHub.Emit(LogEntry{DeviceID: deviceID, Level: "info", Tag: "snapshot", Message: "restored snapshot " + name})
	return nil
}

// WipeData removes /data and /cache but keeps the device and system state.
func (m *Manager) WipeData(deviceID string) error {
	dev, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}
	for _, sub := range []string{"data", "cache", "tmp"} {
		if err := os.RemoveAll(filepath.Join(dev.StorageDir, sub)); err != nil {
			return err
		}
	}
	for _, sub := range []string{"data/apps", "cache", "tmp"} {
		if err := os.MkdirAll(filepath.Join(dev.StorageDir, filepath.FromSlash(sub)), 0o755); err != nil {
			return err
		}
	}
	dev.mu.Lock()
	dev.Installed = map[string]InstalledApp{}
	dev.Running = map[string]AppInstance{}
	err = dev.saveInstalled()
	dev.mu.Unlock()
	if err != nil {
		return err
	}
	m.logHub.Emit(LogEntry{DeviceID: deviceID, Level: "info", Tag: "wipe", Message: "user data wiped"})
	return nil
}

func safeJoin(root, name string) (string, error) {
	cleanRoot := filepath.Clean(root)
	target := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(name)))
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
		return "", ErrAppNotFound
	}
	return target, nil
}
