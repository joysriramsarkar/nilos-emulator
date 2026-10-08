package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/joysriramsarkar/nilos-emulator/internal/qemu"
)

// QEMUStatus is an honest report of the optional System Mode.
type QEMUStatus struct {
	Available bool   `json:"available"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Error     string `json:"error,omitempty"`
	Path      string `json:"path,omitempty"`
	Image     string `json:"image,omitempty"`
	Since     int64  `json:"since,omitempty"`
}

// QEMUBridge manages the optional QEMU instance for full-system (kernel) mode.
type QEMUBridge struct {
	mu       sync.Mutex
	instance *qemu.Instance
	lastErr  string
}

var globalQEMU = &QEMUBridge{}

func osImagePath() string {
	if v := os.Getenv("NILEMU_OS_IMAGE"); v != "" {
		return v
	}
	return "nilemu-system.img"
}

// Status returns the current system-mode status.
func (b *QEMUBridge) Status() QEMUStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.statusLocked()
}

func (b *QEMUBridge) statusLocked() QEMUStatus {
	path := qemu.FindBinary()
	st := QEMUStatus{
		Available: path != "", Path: path, Image: osImagePath(),
		Error: b.lastErr,
	}
	if b.instance != nil {
		st.Running = b.instance.Running()
		st.PID = b.instance.Pid()
		st.ExitCode = b.instance.ExitCode()
		st.Since = b.instance.StartedAt().UnixMilli()
		if err := b.instance.LastError(); err != "" && st.Error == "" {
			st.Error = err
		}
	}
	if !st.Available && st.Error == "" {
		st.Error = "qemu-system-x86_64 was not found; install QEMU or set NILEMU_QEMU"
	}
	return st
}

// Launch starts QEMU with the configured image.
func (b *QEMUBridge) Launch(image string) QEMUStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.instance != nil && b.instance.Running() {
		return b.statusLocked()
	}
	if image == "" {
		image = osImagePath()
	}
	inst, err := qemu.Launch(qemu.LaunchOptions{Image: image, MemoryMB: 512, Accel: true})
	if err != nil {
		b.lastErr = err.Error()
		return b.statusLocked()
	}
	b.instance = inst
	b.lastErr = ""
	return b.statusLocked()
}

// Stop halts the QEMU process.
func (b *QEMUBridge) Stop() QEMUStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.instance != nil {
		_ = b.instance.Stop()
	}
	return b.statusLocked()
}

// Framebuffer captures the real screen as PNG bytes.
func (b *QEMUBridge) Framebuffer() ([]byte, error) {
	b.mu.Lock()
	inst := b.instance
	b.mu.Unlock()
	if inst == nil || !inst.Running() {
		return nil, errNoQEMU
	}
	return inst.ScreenDump()
}

func handleQEMUStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globalQEMU.Status())
}

func handleLaunchQEMU(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	image := r.URL.Query().Get("image")
	json.NewEncoder(w).Encode(globalQEMU.Launch(image))
}

func handleQEMUControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action string `json:"action"`
		Image  string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var st QEMUStatus
	warning := ""
	switch req.Action {
	case "start":
		st = globalQEMU.Launch(req.Image)
	case "stop":
		st = globalQEMU.Stop()
	case "reboot":
		globalQEMU.Stop()
		st = globalQEMU.Launch(req.Image)
	default:
		st = globalQEMU.Status()
		warning = "unknown action"
	}
	json.NewEncoder(w).Encode(map[string]any{"action": req.Action, "status": st, "warning": warning})
}

// handleQEMUFramebuffer streams the real framebuffer as PNG when the VM runs;
// otherwise it reports that system mode is not available.
func handleQEMUFramebuffer(w http.ResponseWriter, r *http.Request) {
	png, err := globalQEMU.Framebuffer()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"available": false,
			"reason":    err.Error(),
			"status":    globalQEMU.Status(),
		})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}

var errNoQEMU = &qemuUnavailable{}

type qemuUnavailable struct{}

func (*qemuUnavailable) Error() string {
	return "system mode is not running; start it from the System panel or /api/qemu/launch"
}
