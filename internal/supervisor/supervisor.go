// Package supervisor ties the virtual device manager, NilLang runtime
// workers, NilHAL and NilBus together. It implements emulator.AppLauncher,
// spawning one worker process (or in-process mock VM) per running app.
package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/bus"
	"github.com/joysriramsarkar/nilos-emulator/internal/config"
	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
	"github.com/joysriramsarkar/nilos-emulator/internal/hal"
	"github.com/joysriramsarkar/nilos-emulator/internal/record"
	"github.com/joysriramsarkar/nilos-emulator/internal/runtime"
	"github.com/joysriramsarkar/nilos-emulator/internal/sandbox"
	"github.com/joysriramsarkar/nilos-emulator/internal/vfs"
)

// Event is streamed to the frontend / DevTools.
type Event struct {
	DeviceID string          `json:"device_id"`
	AppID    string          `json:"app_id"`
	Type     string          `json:"type"`
	Seq      uint64          `json:"seq,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Ts       int64           `json:"ts"`
}

// SessionInfo is a JSON view of a running session.
type SessionInfo struct {
	DeviceID string `json:"device_id"`
	AppID    string `json:"app_id"`
	Session  string `json:"session"`
	State    string `json:"state"`
	Pid      int    `json:"pid"`
	Frames   uint64 `json:"frames"`
	Patches  uint64 `json:"patches"`
}

// Supervisor owns all running app sessions.
type Supervisor struct {
	cfg     config.Config
	manager *emulator.Manager
	bus     *bus.Bus

	mu       sync.RWMutex
	sessions map[string]*session // key: deviceID/appID
	hal      map[string]*hal.Dispatcher
	subs     map[int]chan Event
	nextSub  int

	recordMu     sync.Mutex
	recorder     *record.Recorder
	recordDevice *emulator.Device
}

type session struct {
	info    SessionInfo
	sess    *runtime.Session
	hal     *hal.Dispatcher
	app     emulator.InstalledApp
	device  *emulator.Device
	stopped bool
	release sandbox.Release
}

// New creates a supervisor bound to a device manager.
func New(mgr *emulator.Manager, cfg config.Config) *Supervisor {
	return &Supervisor{
		cfg:      cfg,
		manager:  mgr,
		bus:      bus.New(2000),
		sessions: map[string]*session{},
		hal:      map[string]*hal.Dispatcher{},
		subs:     map[int]chan Event{},
	}
}

// Bus exposes the NilBus instance.
func (s *Supervisor) Bus() *bus.Bus { return s.bus }

// Subscribe streams events to the frontend.
func (s *Supervisor) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 512
	}
	ch := make(chan Event, buffer)
	s.mu.Lock()
	id := s.nextSub
	s.nextSub++
	s.subs[id] = ch
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		if existing, ok := s.subs[id]; ok {
			delete(s.subs, id)
			close(existing)
		}
		s.mu.Unlock()
	}
}

func (s *Supervisor) publish(deviceID, appID, typ string, seq uint64, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	ev := Event{DeviceID: deviceID, AppID: appID, Type: typ, Seq: seq, Data: data, Ts: time.Now().UnixMilli()}
	s.mu.RLock()
	subs := make([]chan Event, 0, len(s.subs))
	for _, ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Sessions lists running sessions.
func (s *Supervisor) Sessions() []SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SessionInfo, 0, len(s.sessions))
	for _, se := range s.sessions {
		info := se.info
		if se.sess != nil {
			info.Frames, info.Patches = se.sess.Stats()
			info.State = se.sess.State()
			info.Pid = se.sess.Pid()
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DeviceID != out[j].DeviceID {
			return out[i].DeviceID < out[j].DeviceID
		}
		return out[i].AppID < out[j].AppID
	})
	return out
}

// HAL returns (creating if needed) the device-level HAL dispatcher.
func (s *Supervisor) HAL(deviceID string) (*hal.Dispatcher, error) {
	s.mu.Lock()
	if d, ok := s.hal[deviceID]; ok {
		s.mu.Unlock()
		return d, nil
	}
	s.mu.Unlock()

	dev, err := s.manager.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	profile, ok := s.manager.Profile(dev.ProfileID)
	if !ok {
		return nil, emulator.ErrInvalidProfile
	}
	root := filepath.Join(dev.StorageDir, "data")
	fsys, err := vfs.New(root)
	if err != nil {
		return nil, err
	}
	d := hal.NewDispatcher(hal.Config{
		DeviceID: deviceID,
		VFS:      fsys,
		WidthPx:  profile.Screen.WidthPx,
		HeightPx: profile.Screen.HeightPx,
		DPR:      profile.Screen.Density,
		Locale:   dev.Settings.Locale,
		ClockISO: s.cfg.ClockISO,
		Freeze:   s.cfg.ClockFrozen,
		OnLog: func(tag, msg string) {
			s.manager.Logs().Emit(emulator.LogEntry{DeviceID: deviceID, Level: "info", Tag: tag, Message: msg})
		},
	})
	s.mu.Lock()
	s.hal[deviceID] = d
	s.mu.Unlock()
	return d, nil
}

// Launch implements emulator.AppLauncher.
func (s *Supervisor) Launch(dev *emulator.Device, app emulator.InstalledApp) (string, int, error) {
	key := dev.ID + "/" + app.ID
	s.mu.Lock()
	if _, exists := s.sessions[key]; exists {
		s.mu.Unlock()
		return "", 0, emulator.ErrAlreadyRunning
	}
	s.mu.Unlock()

	profile, ok := s.manager.Profile(dev.ProfileID)
	if !ok {
		return "", 0, emulator.ErrInvalidProfile
	}

	appInfo := runtime.AppInfo{
		ID:      app.ID,
		Name:    app.Name,
		Version: app.Version,
		Entry:   app.Entry,
		Dir:     app.InstallPath,
	}
	for perm, granted := range app.Permissions {
		if granted {
			appInfo.Permissions = append(appInfo.Permissions, perm)
		}
	}

	devSettings := dev.Settings
	initMsg := runtime.Init{
		Device: runtime.DeviceInit{
			Profile:     profile.ID,
			W:           profile.Screen.WidthPx,
			H:           profile.Screen.HeightPx,
			DPR:         profile.Screen.Density,
			Safe:        runtime.Insets{Top: profile.Screen.SafeArea.Top, Bottom: profile.Screen.SafeArea.Bottom, Left: profile.Screen.SafeArea.Left, Right: profile.Screen.SafeArea.Right},
			RefreshHz:   profile.Screen.RefreshHz,
			Orientation: profile.Screen.Orientation,
			Inputs:      inputList(profile),
		},
		Locale:      devSettings.Locale,
		Theme:       devSettings.Theme,
		FontScale:   devSettings.FontScale,
		Permissions: appInfo.Permissions,
		AppData:     filepath.Join(dev.AppDataDir(app.ID), "data"),
		Seed:        s.cfg.Seed,
	}
	animations := s.cfg.Animations
	initMsg.Animations = &animations

	dispatcher, err := s.HAL(dev.ID)
	if err != nil {
		return "", 0, err
	}
	initMsg.HAL = dispatcher.HardwareInit()

	halFunc := func(devName, op string, args json.RawMessage) (any, error) {
		if !s.permitted(dev.ID, app.ID, devName) {
			return nil, fmt.Errorf("%w: %s", emulator.ErrPermissionDenied, devName)
		}
		args = s.scopeStorage(dev, app.ID, devName, args)
		if r := s.activeRecorder(); r != nil {
			r.Add(record.KindHAL, dev.ID, app.ID, runtime.HalCallMsg{Dev: devName, Op: op, Args: args})
		}
		return dispatcher.Call(devName, op, args)
	}

	cb := runtime.Callbacks{
		OnFrame: func(f runtime.FrameMsg) {
			s.publish(dev.ID, app.ID, "frame", f.Seq, f)
		},
		OnPatch: func(p runtime.PatchMsg) {
			s.publish(dev.ID, app.ID, "patch", p.Seq, p)
		},
		OnLog: func(l runtime.LogMsg) {
			s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: l.Level, Tag: l.Tag, Message: l.Msg})
			s.publish(dev.ID, app.ID, "log", 0, l)
		},
		OnTrace: func(tr runtime.TraceMsg) {
			s.publish(dev.ID, app.ID, "trace", 0, tr)
		},
		OnState: func(st runtime.StateMsg) {
			s.publish(dev.ID, app.ID, "state", 0, st)
		},
		OnBus: func(b runtime.BusMsg) {
			s.bus.Publish(b.Topic, app.ID, b.Payload)
			s.publish(dev.ID, app.ID, "bus", 0, b)
		},
		OnError: func(e runtime.ErrorMsg) {
			s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: "error", Tag: "runtime", Message: e.Msg})
			s.publish(dev.ID, app.ID, "error", 0, e)
		},
		OnExit: func(e runtime.ExitMsg) {
			_ = s.manager.UpdateAppState(dev.ID, app.ID, emulator.LifecycleDestroyed, "", 0)
			s.publish(dev.ID, app.ID, "exit", 0, e)
		},
	}

	transport, pid, err := s.newTransport(dev, app)
	if err != nil {
		return "", 0, err
	}

	// Apply resource limits (Job Object on Windows, cgroup v2 on Linux).
	var release sandbox.Release
	if pid > 0 && (app.Limits.MemoryMB > 0 || app.Limits.CPUPct > 0) {
		release, err = sandbox.Attach(pid, sandbox.Limits{MemoryMB: app.Limits.MemoryMB, CPUPct: app.Limits.CPUPct})
		if err != nil {
			s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: "warn", Tag: "sandbox",
				Message: "could not apply resource limits: " + err.Error()})
		} else if release != nil {
			s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: "info", Tag: "sandbox",
				Message: fmt.Sprintf("limits applied: %dMB / %d%% cpu", app.Limits.MemoryMB, app.Limits.CPUPct)})
		}
	}

	sess, err := runtime.Start(runtime.StartOptions{
		App:       appInfo,
		Transport: transport,
		Init:      initMsg,
		HAL:       halFunc,
		Callbacks: cb,
	})
	if err != nil {
		_ = transport.Close()
		return "", 0, err
	}

	se := &session{info: SessionInfo{DeviceID: dev.ID, AppID: app.ID, Session: sess.ID, State: sess.State(), Pid: pid}, sess: sess, hal: dispatcher, app: app, device: dev, release: release}
	s.mu.Lock()
	s.sessions[key] = se
	s.mu.Unlock()

	_ = s.manager.UpdateAppState(dev.ID, app.ID, emulator.LifecycleStarted, sess.ID, pid)
	s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: "info", Tag: "runtime", Message: "app started"})
	return sess.ID, pid, nil
}

// newTransport chooses a real worker binary when available, otherwise the
// built-in mock VM so the emulator works without the NilLang toolchain.
func (s *Supervisor) newTransport(dev *emulator.Device, app emulator.InstalledApp) (runtime.Transport, int, error) {
	if bin := s.workerBinary(); bin != "" {
		env := append(os.Environ(),
			"NILEMU_DEVICE="+dev.ID,
			"NILEMU_APP="+app.ID,
			"NILEMU_APPDATA="+filepath.Join(dev.AppDataDir(app.ID), "data"),
		)
		tr, err := runtime.NewExecTransport(context.Background(), bin,
			[]string{"--app", app.InstallPath, "--entry", app.Entry, "--id", app.ID, "--name", app.Name},
			app.InstallPath, env, os.Stderr)
		if err == nil {
			return tr, tr.Pid(), nil
		}
		s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: app.ID, Level: "warn", Tag: "runtime",
			Message: "worker binary failed, using built-in mock: " + err.Error()})
	}
	tr := runtime.NewPipeTransport(func(in io.Reader, out io.Writer) error {
		return runtime.RunMockVM(runtime.AppInfo{ID: app.ID, Name: app.Name, Version: app.Version, Entry: app.Entry}, in, out)
	})
	return tr, 0, nil
}

func (s *Supervisor) workerBinary() string {
	if s.cfg.NilosdBin != "" {
		return s.cfg.NilosdBin
	}
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "nilosd"+exeExt())
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// Stop implements emulator.AppLauncher.
func (s *Supervisor) Stop(dev *emulator.Device, appID string) error {
	key := dev.ID + "/" + appID
	s.mu.Lock()
	se, ok := s.sessions[key]
	if ok {
		delete(s.sessions, key)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if se.sess != nil {
		_ = se.sess.Stop()
	}
	if se.release != nil {
		se.release()
	}
	_ = s.manager.UpdateAppState(dev.ID, appID, emulator.LifecycleDestroyed, "", 0)
	s.manager.Logs().Emit(emulator.LogEntry{DeviceID: dev.ID, AppID: appID, Level: "info", Tag: "runtime", Message: "app stopped"})
	return nil
}

// Pause implements emulator.AppLauncher.
func (s *Supervisor) Pause(dev *emulator.Device, appID string) error {
	if se := s.get(dev.ID, appID); se != nil {
		_ = se.sess.SendLifecycle(runtime.LifecyclePaused)
		return s.manager.UpdateAppState(dev.ID, appID, emulator.LifecyclePaused, "", 0)
	}
	return emulator.ErrAppNotFound
}

// Resume implements emulator.AppLauncher.
func (s *Supervisor) Resume(dev *emulator.Device, appID string) error {
	if se := s.get(dev.ID, appID); se != nil {
		_ = se.sess.SendLifecycle(runtime.LifecycleResumed)
		return s.manager.UpdateAppState(dev.ID, appID, emulator.LifecycleResumed, "", 0)
	}
	return emulator.ErrAppNotFound
}

func (s *Supervisor) get(deviceID, appID string) *session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[deviceID+"/"+appID]
}

// SendInput forwards an input event to a running app.
func (s *Supervisor) SendInput(deviceID, appID string, in runtime.InputMsg) error {
	if r := s.activeRecorder(); r != nil {
		r.Add(record.KindInput, deviceID, appID, in)
	}
	se := s.get(deviceID, appID)
	if se == nil {
		return emulator.ErrAppNotFound
	}
	return se.sess.SendInput(in)
}

// ── record / replay ─────────────────────────────────────────────────────

func (s *Supervisor) activeRecorder() *record.Recorder {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	return s.recorder
}

// IsRecording reports whether a capture is in progress.
func (s *Supervisor) IsRecording() bool { return s.activeRecorder() != nil }

// StartRecording begins capturing input and HAL events for an app.
func (s *Supervisor) StartRecording(deviceID, appID, name string) error {
	dev, err := s.manager.GetDevice(deviceID)
	if err != nil {
		return err
	}
	if name == "" {
		name = fmt.Sprintf("rec-%d", time.Now().Unix())
	}
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	if s.recorder != nil {
		return errors.New("a recording is already in progress")
	}
	s.recorder = record.NewRecorder(deviceID, appID, name)
	s.recordDevice = dev
	s.manager.Logs().Emit(emulator.LogEntry{DeviceID: deviceID, AppID: appID, Level: "info", Tag: "record", Message: "recording started: " + name})
	return nil
}

// StopRecording finalises the capture and writes it to the device directory.
func (s *Supervisor) StopRecording() (*record.Recording, string, error) {
	s.recordMu.Lock()
	r := s.recorder
	dev := s.recordDevice
	s.recorder = nil
	s.recordDevice = nil
	s.recordMu.Unlock()
	if r == nil {
		return nil, "", errors.New("no active recording")
	}
	rec := r.Stop()
	path := ""
	if dev != nil {
		dir := filepath.Join(dev.StorageDir, "recordings")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			path = filepath.Join(dir, safeName(rec.Name)+".nilrec")
			if err := r.Save(path); err != nil {
				return &rec, "", err
			}
		}
	}
	s.manager.Logs().Emit(emulator.LogEntry{DeviceID: rec.Device, AppID: rec.App, Level: "info", Tag: "record",
		Message: fmt.Sprintf("recording stopped: %d events", len(rec.Events))})
	return &rec, path, nil
}

// ListRecordings returns the recordings saved for a device.
func (s *Supervisor) ListRecordings(deviceID string) ([]record.Recording, error) {
	dev, err := s.manager.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(dev.StorageDir, "recordings")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []record.Recording{}, nil
	}
	var out []record.Recording
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".nilrec" {
			continue
		}
		if rec, err := record.Load(filepath.Join(dir, e.Name())); err == nil {
			out = append(out, record.Recording{
				Name: rec.Name, CreatedAt: rec.CreatedAt, Device: rec.Device, App: rec.App,
				Events: rec.Events,
			})
		}
	}
	return out, nil
}

// ReplayRecording replays a saved recording into a running app.
func (s *Supervisor) ReplayRecording(deviceID, appID, name string, opts record.PlaybackOptions) error {
	dev, err := s.manager.GetDevice(deviceID)
	if err != nil {
		return err
	}
	rec, err := record.Load(filepath.Join(dev.StorageDir, "recordings", safeName(name)+".nilrec"))
	if err != nil {
		return err
	}
	return s.Replay(deviceID, appID, rec, opts)
}

// Replay feeds a recording's events back into the emulator.
func (s *Supervisor) Replay(deviceID, appID string, rec *record.Recording, opts record.PlaybackOptions) error {
	if appID == "" {
		appID = rec.App
	}
	s.manager.Logs().Emit(emulator.LogEntry{DeviceID: deviceID, AppID: appID, Level: "info", Tag: "record",
		Message: fmt.Sprintf("replaying %d events", len(rec.Events))})
	return record.Play(rec, opts, func(ev record.Event) error {
		switch ev.Kind {
		case record.KindInput:
			var in runtime.InputMsg
			if json.Unmarshal(ev.Data, &in) != nil {
				return nil
			}
			return s.SendInput(deviceID, appID, in)
		case record.KindIME:
			var im runtime.ImeMsg
			if json.Unmarshal(ev.Data, &im) != nil {
				return nil
			}
			return s.SendIME(deviceID, appID, im)
		case record.KindHAL:
			var c runtime.HalCallMsg
			if json.Unmarshal(ev.Data, &c) != nil {
				return nil
			}
			d, err := s.HAL(deviceID)
			if err != nil {
				return err
			}
			_, err = d.Call(c.Dev, c.Op, c.Args)
			return err
		}
		return nil
	})
}

// SendVsync sends a frame credit.
func (s *Supervisor) SendVsync(deviceID, appID string) error {
	if se := s.get(deviceID, appID); se != nil {
		return se.sess.SendVsync()
	}
	return emulator.ErrAppNotFound
}

// SendIME forwards an IME command.
func (s *Supervisor) SendIME(deviceID, appID string, m runtime.ImeMsg) error {
	if r := s.activeRecorder(); r != nil {
		r.Add(record.KindIME, deviceID, appID, m)
	}
	if se := s.get(deviceID, appID); se != nil {
		return se.sess.SendIME(m)
	}
	return emulator.ErrAppNotFound
}

// Reload asks a running app to hot restart.
func (s *Supervisor) Reload(deviceID, appID, mode string) error {
	if se := s.get(deviceID, appID); se != nil {
		if mode == "" {
			mode = "restart"
		}
		return se.sess.SendReload(mode)
	}
	return emulator.ErrAppNotFound
}

// ConfigChange pushes a config update to a running app.
func (s *Supervisor) ConfigChange(deviceID, appID string, c runtime.ConfigMsg) error {
	if se := s.get(deviceID, appID); se != nil {
		return se.sess.SendConfig(c)
	}
	return emulator.ErrAppNotFound
}

// LastTree returns the most recent UI tree for an app.
func (s *Supervisor) LastTree(deviceID, appID string) (json.RawMessage, bool) {
	if se := s.get(deviceID, appID); se != nil {
		return se.sess.LastTree(), true
	}
	return nil, false
}

func (s *Supervisor) permitted(deviceID, appID, devName string) bool {
	perm := permissionForDevice(devName)
	if perm == "" {
		return true
	}
	return s.manager.PermissionGranted(deviceID, appID, perm)
}

// scopeStorage rewrites a storage path so each app writes beneath its own
// private directory within the device data root.
func (s *Supervisor) scopeStorage(dev *emulator.Device, appID, devName string, args json.RawMessage) json.RawMessage {
	if devName != "storage" && devName != "filesystem" && devName != "fs" {
		return args
	}
	var a map[string]any
	if json.Unmarshal(args, &a) != nil {
		return args
	}
	p, _ := a["path"].(string)
	p = strings.TrimPrefix(p, "/")
	a["path"] = "apps/" + appSanitize(appID) + "/" + p
	b, err := json.Marshal(a)
	if err != nil {
		return args
	}
	return b
}

func permissionForDevice(dev string) string {
	switch dev {
	case "battery", "power":
		return "power"
	case "network", "net":
		return "network"
	case "printer":
		return "printer"
	case "scanner":
		return "scanner"
	case "storage", "filesystem", "fs":
		return "filesystem"
	case "sensors":
		return "sensors"
	case "clipboard":
		return "clipboard"
	case "notifications":
		return "notifications"
	case "camera":
		return "camera"
	}
	return ""
}

func inputList(p emulator.DeviceProfile) []string {
	var out []string
	if p.Input.Touch {
		out = append(out, "touch")
	}
	if p.Input.Keyboard {
		out = append(out, "keyboard")
	}
	if p.Input.Mouse {
		out = append(out, "mouse")
	}
	if p.Input.BarcodeScanner {
		out = append(out, "scanner")
	}
	if p.Input.ThermalPrinter {
		out = append(out, "printer")
	}
	return out
}

func appSanitize(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

func exeExt() string {
	if strings.EqualFold(filepath.Ext(os.Args[0]), ".exe") {
		return ".exe"
	}
	return ""
}

// safeName keeps only filename-safe characters so recording names cannot
// escape the recordings directory.
func safeName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "rec"
	}
	return string(out)
}
