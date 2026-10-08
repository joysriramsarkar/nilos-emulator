// Package api exposes the emulator's HTTP control API. It is the same API
// used by the Studio frontend and by the nilctl CLI, mirroring the role of
// the Android emulator console + adb.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/config"
	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
	"github.com/joysriramsarkar/nilos-emulator/internal/hal"
	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
	"github.com/joysriramsarkar/nilos-emulator/internal/record"
	"github.com/joysriramsarkar/nilos-emulator/internal/runtime"
	"github.com/joysriramsarkar/nilos-emulator/internal/supervisor"
)

// Server is the control API handler.
type Server struct {
	cfg     config.Config
	mgr     *emulator.Manager
	sup     *supervisor.Supervisor
	started time.Time
}

// New creates an API server.
func New(cfg config.Config, mgr *emulator.Manager, sup *supervisor.Supervisor) *Server {
	return &Server{cfg: cfg, mgr: mgr, sup: sup, started: time.Now()}
}

// Handler returns the fully wired http.Handler including security middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", s.route)
	return s.security(mux)
}

// security binds the API to loopback origins and enforces an optional token.
func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && !isLoopbackOrigin(origin) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && s.cfg.AuthToken != "" {
			token := r.Header.Get("X-Nilemu-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token != s.cfg.AuthToken {
				http.Error(w, "invalid or missing token", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackOrigin(origin string) bool {
	o := strings.ToLower(origin)
	return strings.HasPrefix(o, "http://127.0.0.1") ||
		strings.HasPrefix(o, "http://localhost") ||
		strings.HasPrefix(o, "http://[::1]") ||
		strings.HasPrefix(o, "https://127.0.0.1")
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	segs := []string{}
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			segs = append(segs, p)
		}
	}
	if len(segs) == 0 {
		s.writeJSON(w, http.StatusOK, map[string]string{"service": "nilemu", "version": "0.2.0"})
		return
	}
	switch segs[0] {
	case "health":
		s.handleHealth(w, r)
	case "profiles":
		s.writeJSON(w, http.StatusOK, s.profilesList())
	case "devices":
		s.handleDevices(w, r, segs[1:])
	case "sessions":
		s.writeJSON(w, http.StatusOK, s.sup.Sessions())
	case "logs":
		n := parseN(r.URL.Query().Get("n"), 200)
		s.writeJSON(w, http.StatusOK, s.mgr.Logs().Recent(n))
	case "bus":
		s.handleBus(w, r)
	case "events":
		s.handleEvents(w, r)
	default:
		s.error(w, http.StatusNotFound, "unknown endpoint")
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"uptime_s": int(time.Since(s.started).Seconds()),
		"data_dir": s.mgr.DataDir(),
		"mode":     "runtime",
	})
}

func (s *Server) profilesList() []map[string]any {
	profiles := s.mgr.Profiles()
	out := make([]map[string]any, 0, len(profiles))
	for id, p := range profiles {
		out = append(out, map[string]any{"id": id, "profile": p})
	}
	return out
}

// ─── devices ────────────────────────────────────────────────────────────

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request, segs []string) {
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.writeJSON(w, http.StatusOK, s.mgr.ListDevices())
		case http.MethodPost:
			var req struct {
				ProfileID string `json:"profile_id"`
				Name      string `json:"name"`
			}
			if !s.decode(w, r, &req) {
				return
			}
			if req.ProfileID == "" {
				s.error(w, http.StatusBadRequest, "profile_id is required")
				return
			}
			dev, err := s.mgr.CreateDevice(req.ProfileID, req.Name)
			if err != nil {
				s.error(w, http.StatusBadRequest, err.Error())
				return
			}
			s.writeJSON(w, http.StatusCreated, dev)
		default:
			s.error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	id := segs[0]
	dev, err := s.mgr.GetDevice(id)
	if err != nil {
		s.error(w, http.StatusNotFound, err.Error())
		return
	}

	if len(segs) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.writeJSON(w, http.StatusOK, s.deviceView(dev))
		case http.MethodDelete:			if err := s.mgr.DeleteDevice(id); err != nil {
				s.error(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
		default:
			s.error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch segs[1] {
	case "start":
		s.okOrErr(w, s.mgr.StartDevice(r.Context(), id))
	case "stop":
		s.okOrErr(w, s.mgr.StopDevice(id))
	case "wipe":
		s.okOrErr(w, s.mgr.WipeData(id))
	case "apps":
		s.handleApps(w, r, dev, segs[2:])
	case "settings":
		s.handleSettings(w, r, id)
	case "battery":
		s.handleBattery(w, r, id)
	case "network":
		s.handleNetwork(w, r, id)
	case "install":
		s.handleInstall(w, r, id)
	case "snapshots":
		s.handleSnapshots(w, r, id, segs[2:])
	case "logs":
		s.writeJSON(w, http.StatusOK, s.filterLogs(id))
	case "hal":
		s.handleHAL(w, r, id, segs[2:])
	case "record":
		s.handleRecord(w, r, id, segs[2:])
	case "recordings":
		s.handleRecordings(w, r, id, segs[2:])
	case "input":
		s.handleDeviceInput(w, r, id, dev)
	case "tree":
		s.handleTree(w, r, id, "")
	default:
		s.error(w, http.StatusNotFound, "unknown device subresource")
	}
}

func (s *Server) deviceView(dev *emulator.Device) map[string]any {
	profile, _ := s.mgr.Profile(dev.ProfileID)
	installed, _ := s.mgr.ListInstalled(dev.ID)
	info, _ := s.mgr.DeviceInfo(dev.ID)
	return map[string]any{
		"info":      info,
		"profile":   profile,
		"installed": installed,
	}
}

// ─── apps ───────────────────────────────────────────────────────────────

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request, dev *emulator.Device, segs []string) {
	if len(segs) == 0 {
		installed, err := s.mgr.ListInstalled(dev.ID)
		if err != nil {
			s.error(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, installed)
		return
	}
	appID := segs[0]
	if len(segs) == 1 {
		app, err := s.mgr.GetInstalled(dev.ID, appID)
		if err != nil {
			s.error(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, app)
		return
	}
	action := segs[1]
	switch action {
	case "start":
		if _, _, err := s.sup.Launch(dev, s.mustApp(dev, appID)); err != nil {
			s.error(w, http.StatusBadRequest, err.Error())
			return
		}
		s.startAppAuto(dev, appID)
		s.writeJSON(w, http.StatusOK, map[string]any{"started": true})
	case "stop":
		app := s.mustApp(dev, appID)
		s.okOrErr(w, s.sup.Stop(dev, app.ID))
	case "pause":
		app := s.mustApp(dev, appID)
		s.okOrErr(w, s.sup.Pause(dev, app.ID))
	case "resume":
		app := s.mustApp(dev, appID)
		s.okOrErr(w, s.sup.Resume(dev, app.ID))
	case "reload":
		var req struct {
			Mode string `json:"mode"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		s.okOrErr(w, s.sup.Reload(dev.ID, appID, req.Mode))
	case "uninstall":
		s.okOrErr(w, s.mgr.Uninstall(dev.ID, appID))
	case "clear-data":
		s.okOrErr(w, s.mgr.ClearAppData(dev.ID, appID))
	case "permissions":
		s.handlePermissions(w, r, dev.ID, appID)
	case "input":
		s.handleAppInput(w, r, dev.ID, appID)
	case "ime":
		s.handleIME(w, r, dev.ID, appID)
	case "tree":
		s.handleTree(w, r, dev.ID, appID)
	case "debug":
		s.handleAppDebug(w, r, dev.ID, appID, segs[2:])
	default:
		s.error(w, http.StatusNotFound, "unknown app action")
	}
}

func (s *Server) mustApp(dev *emulator.Device, appID string) emulator.InstalledApp {
	app, err := s.mgr.GetInstalled(dev.ID, appID)
	if err != nil {
		return emulator.InstalledApp{ID: appID, Name: appID}
	}
	return app
}

// startAppAuto sends an initial vsync so the app can present a frame.
func (s *Server) startAppAuto(dev *emulator.Device, appID string) {
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.sup.SendVsync(dev.ID, appID)
	}()
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request, deviceID, appID string) {
	if appID == "" {
		// pick the first running app for the device
		for _, se := range s.sup.Sessions() {
			if se.DeviceID == deviceID {
				appID = se.AppID
				break
			}
		}
	}
	tree, ok := s.sup.LastTree(deviceID, appID)
	if !ok {
		s.writeJSON(w, http.StatusOK, map[string]any{"tree": nil, "app": appID})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(tree)
}

// ─── settings / hal ─────────────────────────────────────────────────────

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request, id string) {
	var req emulator.DeviceSettings
	if !s.decode(w, r, &req) {
		return
	}
	err := s.mgr.SetSetting(id, func(dst *emulator.DeviceSettings) {
		dst.Theme = req.Theme
		dst.Locale = req.Locale
		dst.FontScale = req.FontScale
		dst.AnimationScale = req.AnimationScale
		dst.WifiEnabled = req.WifiEnabled
		dst.AirplaneMode = req.AirplaneMode
		dst.BatteryLevel = req.BatteryLevel
		dst.Charging = req.Charging
		dst.NetworkMode = req.NetworkMode
		dst.Volume = req.Volume
		dst.Brightness = req.Brightness
		dst.DeveloperMode = req.DeveloperMode
	})
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	// push config to running apps
	for _, se := range s.sup.Sessions() {
		if se.DeviceID == id {
			_ = s.sup.ConfigChange(id, se.AppID, runtime.ConfigMsg{Theme: req.Theme, Locale: req.Locale, FontScale: req.FontScale})
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleBattery(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Level    int  `json:"level"`
		Charging bool `json:"charging"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	d, err := s.sup.HAL(id)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := d.SetBattery(req.Level, req.Charging)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.mgr.SetSetting(id, func(dst *emulator.DeviceSettings) {
		dst.BatteryLevel = req.Level
		dst.Charging = req.Charging
	})
	s.writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Mode string `json:"mode"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	d, err := s.sup.HAL(id)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := d.SetNetwork(req.Mode)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.mgr.SetSetting(id, func(dst *emulator.DeviceSettings) { dst.NetworkMode = req.Mode })
	s.writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleHAL(w http.ResponseWriter, r *http.Request, id string, segs []string) {
	d, err := s.sup.HAL(id)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(segs) == 0 && r.Method == http.MethodGet {
		s.writeJSON(w, http.StatusOK, d.Snapshot())
		return
	}
	if len(segs) == 1 && segs[0] == "fault" && r.Method == http.MethodPost {
		var req struct {
			Dev   string `json:"dev"`
			Op    string `json:"op"`
			Code  string `json:"code"`
			Times int    `json:"times"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		d.InjectFault(hal.FaultRule{Dev: req.Dev, Op: req.Op, Code: req.Code, Times: req.Times})
		s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if len(segs) == 1 && segs[0] == "call" && r.Method == http.MethodPost {
		var req struct {
			Dev  string          `json:"dev"`
			Op   string          `json:"op"`
			Args json.RawMessage `json:"args"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		res, err := d.Call(req.Dev, req.Op, req.Args)
		if err != nil {
			s.error(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": res})
		return
	}
	if len(segs) == 1 && segs[0] == "faults" && r.Method == http.MethodDelete {
		d.ClearFaults()
		s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	s.error(w, http.StatusNotFound, "unknown HAL action")
}

// ─── input ──────────────────────────────────────────────────────────────

func (s *Server) handleDeviceInput(w http.ResponseWriter, r *http.Request, id string, dev *emulator.Device) {
	appID := r.URL.Query().Get("app")
	if appID == "" {
		for _, se := range s.sup.Sessions() {
			if se.DeviceID == id {
				appID = se.AppID
				break
			}
		}
	}
	s.forwardInput(w, r, id, appID)
}

func (s *Server) handleAppInput(w http.ResponseWriter, r *http.Request, deviceID, appID string) {
	s.forwardInput(w, r, deviceID, appID)
}

func (s *Server) forwardInput(w http.ResponseWriter, r *http.Request, deviceID, appID string) {
	var req struct {
		Kind  string  `json:"kind"`
		Phase string  `json:"phase"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		Code  string  `json:"code"`
		Key   string  `json:"key"`
		Text  string  `json:"text"`
		ID    int     `json:"id"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	if appID == "" {
		s.error(w, http.StatusNotFound, "no running app")
		return
	}
	err := s.sup.SendInput(deviceID, appID, runtime.InputMsg{
		Kind: req.Kind, Phase: req.Phase, X: req.X, Y: req.Y,
		Code: req.Code, Key: req.Key, Text: req.Text, ID: req.ID,
	})
	s.okOrErr(w, err)
}

func (s *Server) handleIME(w http.ResponseWriter, r *http.Request, deviceID, appID string) {
	var req struct {
		Op   string `json:"op"`
		Text string `json:"text"`
		N    int    `json:"n"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	s.okOrErr(w, s.sup.SendIME(deviceID, appID, runtime.ImeMsg{Op: req.Op, Text: req.Text, N: req.N}))
}

// ─── record / replay ────────────────────────────────────────────────────

func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request, id string, segs []string) {
	if len(segs) == 0 || segs[0] == "status" {
		s.writeJSON(w, http.StatusOK, map[string]bool{"recording": s.sup.IsRecording()})
		return
	}
	switch segs[0] {
	case "start":
		var req struct {
			App  string `json:"app"`
			Name string `json:"name"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		s.okOrErr(w, s.sup.StartRecording(id, req.App, req.Name))
	case "stop":
		rec, path, err := s.sup.StopRecording()
		if err != nil {
			s.error(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"name": rec.Name, "events": len(rec.Events), "path": path,
			"device": rec.Device, "app": rec.App,
		})
	default:
		s.error(w, http.StatusNotFound, "unknown record action")
	}
}

func (s *Server) handleRecordings(w http.ResponseWriter, r *http.Request, id string, segs []string) {
	if len(segs) == 0 {
		if r.Method != http.MethodGet {
			s.error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		list, err := s.sup.ListRecordings(id)
		if err != nil {
			s.error(w, http.StatusNotFound, err.Error())
			return
		}
		summaries := make([]map[string]any, 0, len(list))
		for _, rec := range list {
			summaries = append(summaries, map[string]any{
				"name": rec.Name, "created_at": rec.CreatedAt,
				"device": rec.Device, "app": rec.App, "events": len(rec.Events),
			})
		}
		s.writeJSON(w, http.StatusOK, summaries)
		return
	}
	name := segs[0]
	if len(segs) == 2 && segs[1] == "replay" && r.Method == http.MethodPost {
		var req struct {
			App     string  `json:"app"`
			Instant bool    `json:"instant"`
			Speed   float64 `json:"speed"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err := s.sup.ReplayRecording(id, req.App, name, record.PlaybackOptions{Speed: req.Speed, Instant: req.Instant})
		s.okOrErr(w, err)
		return
	}
	s.error(w, http.StatusNotFound, "unknown recording action")
}

// ─── permissions ────────────────────────────────────────────────────────

func (s *Server) handlePermissions(w http.ResponseWriter, r *http.Request, deviceID, appID string) {
	switch r.Method {
	case http.MethodGet:
		app, err := s.mgr.GetInstalled(deviceID, appID)
		if err != nil {
			s.error(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, app.Permissions)
	case http.MethodPost:
		var req struct {
			Permission string `json:"permission"`
			Grant      bool   `json:"grant"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		var err error
		if req.Grant {
			err = s.mgr.GrantPermission(deviceID, appID, req.Permission)
		} else {
			err = s.mgr.RevokePermission(deviceID, appID, req.Permission)
		}
		s.okOrErr(w, err)
	default:
		s.error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ─── install / snapshots / logs / bus / events ──────────────────────────

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := ""
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(128 << 20); err != nil {
			s.error(w, http.StatusBadRequest, err.Error())
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			s.error(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()
		tmpDir := filepath.Join(s.mgr.DataDir(), "tmp")
		_ = os.MkdirAll(tmpDir, 0o755)
		path = filepath.Join(tmpDir, filepath.Base(hdr.Filename))
		out, err := os.Create(path)
		if err != nil {
			s.error(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer os.Remove(path)
		if _, err := io.Copy(out, file); err != nil {
			out.Close()
			s.error(w, http.StatusInternalServerError, err.Error())
			return
		}
		out.Close()
	} else {
		var req struct {
			Path string `json:"path"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		path = req.Path
	}
	if path == "" {
		s.error(w, http.StatusBadRequest, "bundle path is required")
		return
	}
	opts := packages.DefaultOptions()
	opts.TrustKeys = packages.TrustKeysFromHex(s.cfg.TrustedKeys)
	opts.AllowUnsigned = s.cfg.AllowUnsigned
	app, err := s.mgr.InstallBundle(id, path, opts)
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeJSON(w, http.StatusCreated, app)
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request, id string, segs []string) {
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodGet:
			list, _ := s.mgr.ListSnapshots(id)
			s.writeJSON(w, http.StatusOK, list)
		case http.MethodPost:
			var req struct {
				Name string `json:"name"`
			}
			if !s.decode(w, r, &req) {
				return
			}
			snap, err := s.mgr.SaveSnapshot(id, req.Name)
			if err != nil {
				s.error(w, http.StatusBadRequest, err.Error())
				return
			}
			s.writeJSON(w, http.StatusCreated, snap)
		default:
			s.error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	name := segs[0]
	if len(segs) == 2 && segs[1] == "restore" && r.Method == http.MethodPost {
		s.okOrErr(w, s.mgr.RestoreSnapshot(id, name))
		return
	}
	s.error(w, http.StatusNotFound, "unknown snapshot action")
}

func (s *Server) filterLogs(deviceID string) []emulator.LogEntry {
	recent := s.mgr.Logs().Recent(0)
	out := make([]emulator.LogEntry, 0, len(recent))
	for _, e := range recent {
		if e.DeviceID == deviceID {
			out = append(out, e)
		}
	}
	return out
}

func (s *Server) handleBus(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/log"):
		s.writeJSON(w, http.StatusOK, s.sup.Bus().Log(parseN(r.URL.Query().Get("n"), 200)))
	case strings.HasSuffix(r.URL.Path, "/publish") && r.Method == http.MethodPost:
		var req struct {
			Topic   string          `json:"topic"`
			From    string          `json:"from"`
			Payload json.RawMessage `json:"payload"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		s.writeJSON(w, http.StatusOK, s.sup.Bus().Publish(req.Topic, req.From, req.Payload))
	default:
		s.error(w, http.StatusNotFound, "unknown bus endpoint")
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.error(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	deviceID := r.URL.Query().Get("device")
	appID := r.URL.Query().Get("app")
	ch, cancel := s.sup.Subscribe(2048)
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if deviceID != "" && ev.DeviceID != deviceID {
				continue
			}
			if appID != "" && ev.AppID != appID {
				continue
			}
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ─── debug ──────────────────────────────────────────────────────────────

func (s *Server) handleAppDebug(w http.ResponseWriter, r *http.Request, deviceID, appID string, segs []string) {
	if len(segs) == 0 {
		s.error(w, http.StatusNotFound, "unknown debug endpoint")
		return
	}
	switch segs[0] {
	case "stack":
		s.writeJSON(w, http.StatusOK, map[string]any{"stackFrames": s.sup.DebugStack(deviceID, appID)})
	case "scopes":
		s.writeJSON(w, http.StatusOK, map[string]any{"scopes": s.sup.DebugScopes(deviceID, appID)})
	case "variables":
		ref := parseN(r.URL.Query().Get("ref"), 1)
		s.writeJSON(w, http.StatusOK, map[string]any{"variables": s.sup.DebugVariables(deviceID, appID, ref)})
	case "breakpoints":
		var req struct {
			Source string `json:"source"`
			Lines  []int  `json:"lines"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"breakpoints": s.sup.SetBreakpoints(deviceID, appID, req.Source, req.Lines)})
	case "command":
		var req struct {
			Command string `json:"command"`
		}
		if !s.decode(w, r, &req) {
			return
		}
		s.okOrErr(w, s.sup.DebugCommand(deviceID, appID, req.Command))
	case "evaluate":
		result, err := s.sup.DebugEvaluate(deviceID, appID, r.URL.Query().Get("expr"))
		if err != nil {
			s.error(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]string{"result": result})
	default:
		s.error(w, http.StatusNotFound, "unknown debug action")
	}
}

// ─── helpers ────────────────────────────────────────────────────────────

func (s *Server) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		s.error(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) error(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]any{"error": msg, "status": status})
}

func (s *Server) okOrErr(w http.ResponseWriter, err error) {
	if err != nil {
		s.error(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseN(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
