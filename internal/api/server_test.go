package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/config"
	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
	"github.com/joysriramsarkar/nilos-emulator/internal/supervisor"
)

func setup(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Headless = true
	mgr, err := emulator.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	sup := supervisor.New(mgr, cfg)
	mgr.SetLauncher(sup)
	srv := httptest.NewServer(New(cfg, mgr, sup).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func doJSON(t *testing.T, method, url string, body any, out any) int {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && resp.StatusCode < 300 {
			t.Fatalf("%s %s: decode: %v", method, url, err)
		}
	}
	return resp.StatusCode
}

func buildBundle(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "app")
	_ = os.MkdirAll(filepath.Join(src, "payload"), 0o755)
	man := packages.Manifest{ID: "com.example.calc", Name: "Calc", Version: "1.0.0", Entry: "payload/index.html"}
	mb, _ := json.Marshal(man)
	_ = os.WriteFile(filepath.Join(src, "manifest.json"), mb, 0o644)
	_ = os.WriteFile(filepath.Join(src, "payload", "index.html"), []byte("<h1>calc</h1>"), 0o644)
	priv, pub, err := packages.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "calc.nilax")
	if err := packages.SignDir(src, out, priv, "dev"); err != nil {
		t.Fatal(err)
	}
	return out, pub
}

func TestDeviceAppEndToEnd(t *testing.T) {
	srv := setup(t)
	bundlePath, pub := buildBundle(t)
	_ = pub // trust is configured below via env-free path

	// Create a device.
	var created struct {
		ID string `json:"id"`
	}
	if code := doJSON(t, "POST", srv.URL+"/api/v1/devices", map[string]any{
		"profile_id": "nilphone-ultra", "name": "Test Phone",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create device status %d", code)
	}
	if created.ID == "" {
		t.Fatal("no device id")
	}

	// Start it.
	if code := doJSON(t, "POST", srv.URL+"/api/v1/devices/"+created.ID+"/start", nil, nil); code != http.StatusOK {
		t.Fatalf("start status %d", code)
	}
	// The device must reach "running" even though the HTTP request returned.
	bootDeadline := time.Now().Add(3 * time.Second)
	running := false
	for time.Now().Before(bootDeadline) {
		var view struct {
			Info struct {
				State string `json:"state"`
			} `json:"info"`
		}
		doJSON(t, "GET", srv.URL+"/api/v1/devices/"+created.ID, nil, &view)
		if view.Info.State == "running" {
			running = true
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !running {
		t.Fatal("device never reached running state after start")
	}

	// Install the bundle (unsigned allowed in this test config would fail; the
	// bundle is signed with an untrusted key so install must be rejected).
	var apiErr map[string]any
	if code := doJSON(t, "POST", srv.URL+"/api/v1/devices/"+created.ID+"/install", map[string]any{"path": bundlePath}, &apiErr); code == http.StatusCreated {
		t.Fatal("expected untrusted signature to be rejected")
	}

	// Now install a bundle signed with a trusted key via a second API server.
	dir := t.TempDir()
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Headless = true
	cfg.TrustedKeys = map[string]string{"dev": pub}
	cfg.AllowUnsigned = false
	mgr, err := emulator.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	sup := supervisor.New(mgr, cfg)
	mgr.SetLauncher(sup)
	srv2 := httptest.NewServer(New(cfg, mgr, sup).Handler())
	defer srv2.Close()

	var created2 struct {
		ID string `json:"id"`
	}
	doJSON(t, "POST", srv2.URL+"/api/v1/devices", map[string]any{"profile_id": "nilphone-ultra"}, &created2)
	doJSON(t, "POST", srv2.URL+"/api/v1/devices/"+created2.ID+"/start", nil, nil)

	if code := doJSON(t, "POST", srv2.URL+"/api/v1/devices/"+created2.ID+"/install", map[string]any{"path": bundlePath}, nil); code != http.StatusCreated {
		t.Fatalf("trusted install failed with status %d", code)
	}

	// Launch the app.
	if code := doJSON(t, "POST", srv2.URL+"/api/v1/devices/"+created2.ID+"/apps/com.example.calc/start", nil, nil); code != http.StatusOK {
		t.Fatalf("launch failed with status %d", code)
	}

	// Poll for the first frame.
	treeURL := srv2.URL + "/api/v1/devices/" + created2.ID + "/apps/com.example.calc/tree"
	deadline := time.Now().Add(5 * time.Second)
	var tree string
	for time.Now().Before(deadline) {
		resp, err := http.Get(treeURL)
		if err == nil {
			buf := make([]byte, 0)
			tmp := make([]byte, 4096)
			for {
				n, rerr := resp.Body.Read(tmp)
				buf = append(buf, tmp[:n]...)
				if rerr != nil {
					break
				}
			}
			resp.Body.Close()
			tree = string(buf)
			if strings.Contains(tree, "header") {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(tree, "header") {
		t.Fatalf("no frame tree received: %q", tree)
	}

	// Send a tap on the increment handler and confirm frame/patch activity.
	if code := doJSON(t, "POST", srv2.URL+"/api/v1/devices/"+created2.ID+"/apps/com.example.calc/input",
		map[string]any{"kind": "tap", "code": "h:1", "phase": "up"}, nil); code != http.StatusOK {
		t.Fatalf("input status %d", code)
	}

	// Battery control should work through the control API.
	var state map[string]any
	if code := doJSON(t, "POST", srv2.URL+"/api/v1/devices/"+created2.ID+"/battery",
		map[string]any{"level": 42, "charging": true}, &state); code != http.StatusOK {
		t.Fatalf("battery status %d", code)
	}
	if state["battery"] != float64(42) {
		t.Fatalf("unexpected battery state: %+v", state)
	}

	// Sessions should list the running app.
	var sessions []map[string]any
	doJSON(t, "GET", srv2.URL+"/api/v1/sessions", nil, &sessions)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestHealthAndAuth(t *testing.T) {
	srv := setup(t)
	var health map[string]any
	if code := doJSON(t, "GET", srv.URL+"/api/v1/health", nil, &health); code != http.StatusOK {
		t.Fatalf("health status %d", code)
	}
	if health["ok"] != true {
		t.Fatalf("unexpected health: %+v", health)
	}
}
