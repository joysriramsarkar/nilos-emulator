package emulator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func buildTestBundle(t *testing.T) (path, pubHex string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "app")
	if err := os.MkdirAll(filepath.Join(src, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := packages.Manifest{
		ID: "com.example.notes", Name: "Notes", Version: "1.0.0",
		Entry: "payload/index.html", Permissions: []string{"filesystem", "printer"},
	}
	mb, _ := json.Marshal(man)
	_ = os.WriteFile(filepath.Join(src, "manifest.json"), mb, 0o644)
	_ = os.WriteFile(filepath.Join(src, "payload", "index.html"), []byte("<h1>notes</h1>"), 0o644)
	priv, pub, err := packages.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "notes.nilax")
	if err := packages.SignDir(src, out, priv, "test"); err != nil {
		t.Fatal(err)
	}
	return out, pub
}

func TestDeviceLifecycleAndInstall(t *testing.T) {
	m := newTestManager(t)
	info, err := m.CreateDevice("nilphone-ultra", "Pixel Nil")
	if err != nil {
		t.Fatal(err)
	}
	if info.State != StateOff {
		t.Fatalf("expected off, got %s", info.State)
	}
	if len(m.ListDevices()) != 1 {
		t.Fatal("expected one device")
	}

	if err := m.StartDevice(context.Background(), info.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		dev, _ := m.GetDevice(info.ID)
		dev.mu.Lock()
		state := dev.State
		dev.mu.Unlock()
		if state == StateRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("device did not reach running, state=%s", state)
		}
		time.Sleep(20 * time.Millisecond)
	}

	bundlePath, pub := buildTestBundle(t)
	opts := packages.DefaultOptions()
	opts.TrustKeys = packages.TrustKeysFromHex(map[string]string{"test": pub})
	app, err := m.InstallBundle(info.ID, bundlePath, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if app.ID != "com.example.notes" || !app.Signed {
		t.Fatalf("unexpected app: %+v", app)
	}
	if !app.Permissions["printer"] {
		t.Fatal("declared permission not granted")
	}

	if err := m.RevokePermission(info.ID, app.ID, "printer"); err != nil {
		t.Fatal(err)
	}
	if m.PermissionGranted(info.ID, app.ID, "printer") {
		t.Fatal("expected printer permission revoked")
	}

	installed, _ := m.ListInstalled(info.ID)
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed app, got %d", len(installed))
	}

	snap, err := m.SaveSnapshot(info.ID, "clean")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Name != "clean" {
		t.Fatal("unexpected snapshot")
	}
	snaps, _ := m.ListSnapshots(info.ID)
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	if err := m.RestoreSnapshot(info.ID, "clean"); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if err := m.ClearAppData(info.ID, app.ID); err != nil {
		t.Fatal(err)
	}

	if err := m.Uninstall(info.ID, app.ID); err != nil {
		t.Fatal(err)
	}
	installed, _ = m.ListInstalled(info.ID)
	if len(installed) != 0 {
		t.Fatal("expected app uninstalled")
	}

	if err := m.WipeData(info.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.StopDevice(info.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteDevice(info.ID); err != nil {
		t.Fatal(err)
	}
	if len(m.ListDevices()) != 0 {
		t.Fatal("expected no devices")
	}
}

func TestUnknownProfile(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.CreateDevice("nonexistent", ""); err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestDevicePersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	m1, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	info, err := m1.CreateDevice("nilpos-terminal", "Persisted")
	if err != nil {
		t.Fatal(err)
	}

	m2, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.ListDevices()) != 1 {
		t.Fatal("device not reloaded from disk")
	}
	got, err := m2.DeviceInfo(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProfileID != "nilpos-terminal" {
		t.Fatalf("profile_id lost after reload: %q", got.ProfileID)
	}
	if got.Name != "Persisted" {
		t.Fatalf("name lost after reload: %q", got.Name)
	}
}
