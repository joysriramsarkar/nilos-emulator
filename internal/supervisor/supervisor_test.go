package supervisor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/config"
	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
	"github.com/joysriramsarkar/nilos-emulator/internal/runtime"
)

func buildSup(t *testing.T) (*Supervisor, *emulator.Manager, *emulator.Device, emulator.InstalledApp) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := emulator.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.DataDir = dir
	sup := New(mgr, cfg)
	mgr.SetLauncher(sup)

	info, err := mgr.CreateDevice("nilpos-terminal", "SupPOS")
	if err != nil {
		t.Fatal(err)
	}
	dev, _ := mgr.GetDevice(info.ID)

	appDir := filepath.Join(dev.StorageDir, "data", "apps", "com.example.pos")
	if err := os.MkdirAll(filepath.Join(appDir, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := emulator.InstalledApp{
		ID: "com.example.pos", Name: "POS", Version: "1.0.0",
		Entry: "payload/index.html", InstallPath: appDir, Enabled: true,
		Permissions: map[string]bool{"printer": true, "filesystem": true},
	}
	dev.Installed[app.ID] = app
	return sup, mgr, dev, app
}

func TestLaunchPauseResumeStop(t *testing.T) {
	sup, _, dev, app := buildSup(t)

	frames := make(chan Event, 32)
	_, cancel := sup.Subscribe(64)
	events, cancel2 := sup.Subscribe(64)
	defer cancel()
	defer cancel2()
	go func() {
		for ev := range events {
			if ev.Type == "frame" {
				frames <- ev
			}
		}
	}()

	id, _, err := sup.Launch(dev, app)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if id == "" {
		t.Fatal("empty session id")
	}

	select {
	case <-frames:
	case <-time.After(3 * time.Second):
		t.Fatal("no frame event")
	}

	if err := sup.Pause(dev, app.ID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := sup.Resume(dev, app.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := sup.Reload(dev.ID, app.ID, "restart"); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := sup.Stop(dev, app.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(sup.Sessions()) != 0 {
		t.Fatal("expected no sessions after stop")
	}
}

func TestPermissionDeniedForPrinter(t *testing.T) {
	sup, mgr, dev, app := buildSup(t)
	// Revoke printer so the HAL guard must deny it.
	if err := mgr.RevokePermission(dev.ID, app.ID, "printer"); err != nil {
		t.Fatal(err)
	}

	d, err := sup.HAL(dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the guard used by the supervisor.
	if sup.permitted(dev.ID, app.ID, "printer") {
		t.Fatal("expected printer to be denied")
	}
	if !sup.permitted(dev.ID, app.ID, "filesystem") {
		t.Fatal("expected filesystem to be allowed")
	}
	// Direct dispatcher call still works (guard is in the supervisor layer).
	if _, err := d.Call("printer", "print", json.RawMessage(`{"lines":["x"]}`)); err != nil {
		t.Fatalf("dispatcher print failed: %v", err)
	}
}

func TestHALDispatcherSharedPerDevice(t *testing.T) {
	sup, _, dev, _ := buildSup(t)
	d1, err := sup.HAL(dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	d2, _ := sup.HAL(dev.ID)
	if d1 != d2 {
		t.Fatal("expected HAL dispatcher to be cached per device")
	}
}

var _ = runtime.ProtoVersion
var _ = packages.DefaultOptions
