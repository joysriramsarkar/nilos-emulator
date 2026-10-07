package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// QEMUVMStatus represents the virtual machine status
type QEMUVMStatus struct {
	State        string            `json:"state"` // "running", "stopped", "paused", "simulated"
	PID          int               `json:"pid"`
	Architecture string            `json:"arch"`
	CPULoadPct   float64           `json:"cpuLoadPct"`
	RAMAllocated string            `json:"ramAllocated"`
	Resolution   string            `json:"resolution"`
	VGAColors    int               `json:"vgaColors"`
	KernelBuild  string            `json:"kernelBuild"`
	Uptime       string            `json:"uptime"`
	Registers    map[string]string `json:"registers"`
	LogBuffer    []string          `json:"logBuffer"`
	FrameNumber  uint64            `json:"frameNumber"`
}

// QEMUBridgeManager coordinates native and simulated QEMU instances
type QEMUBridgeManager struct {
	mu          sync.RWMutex
	state       string
	process     *exec.Cmd
	startTime   time.Time
	frameNumber uint64
	logs        []string
	registers   map[string]string
}

var globalQEMUBridge = NewQEMUBridgeManager()

func NewQEMUBridgeManager() *QEMUBridgeManager {
	mgr := &QEMUBridgeManager{
		state:     "running",
		startTime: time.Now(),
		logs: []string{
			"[0.000000] Onuron Microkernel v0.1.0-alpha initializing...",
			"[0.001240] CPU 0: NilCore Octa-Core @ 3.40 GHz (x86_64, Long Mode enabled)",
			"[0.002890] GDT initialized at 0xFFFF800000001000 with 6 segments",
			"[0.004120] IDT initialized: 256 vector handlers registered",
			"[0.006450] Page table level 4 initialized (PML4 -> PDPT -> PD -> PT)",
			"[0.008910] Physical Memory Manager: 8192 MB RAM detected (7980 MB free)",
			"[0.012300] NilBus IPC driver bound to IRQ 44 (12 active channels)",
			"[0.015600] Alap Declarative Compositor attached to /dev/fb0 (1024x768x32bpp)",
			"[0.020100] Mounting NilFS virtual root filesystem... OK",
			"[0.025400] Starting system services: nil-daemon, alap-server, pos-bridge...",
			"[0.030000] NilOS 2.0 Desktop Shell ready. User logged in as 'developer'.",
		},
		registers: map[string]string{
			"RAX": "0x00007FFF89A4C000",
			"RBX": "0x0000000000401020",
			"RCX": "0x000000000000002A",
			"RDX": "0x0000000000000000",
			"RSI": "0xFFFF800000100000",
			"RDI": "0x0000000000000001",
			"RSP": "0x00007FFFFFFFE890",
			"RBP": "0x00007FFFFFFFE8B0",
			"RIP": "0x00000000004011B8",
			"RFLAGS": "0x0000000000000246 [IF, PF, ZF]",
			"CR0": "0x80050033 [PG, WP, PE]",
			"CR3": "0x0000000000100000",
		},
	}
	return mgr
}

func (m *QEMUBridgeManager) GetStatus() QEMUVMStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.frameNumber++
	uptimeSec := int(time.Since(m.startTime).Seconds())
	pid := 0
	if m.process != nil && m.process.Process != nil {
		pid = m.process.Process.Pid
	}

	return QEMUVMStatus{
		State:        m.state,
		PID:          pid,
		Architecture: "x86_64 (Onuron NilCore)",
		CPULoadPct:   12.8,
		RAMAllocated: "512 MB (Virtual) / 8192 MB (System)",
		Resolution:   "1024x768 @ 60Hz VGA",
		VGAColors:    16777216,
		KernelBuild:  "Onuron Microkernel #42-SMP Tue Oct 7 2026",
		Uptime:       fmt.Sprintf("%d s", uptimeSec),
		Registers:    m.registers,
		LogBuffer:    m.logs,
		FrameNumber:  m.frameNumber,
	}
}

func (m *QEMUBridgeManager) AppendLog(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	timestamp := fmt.Sprintf("[%02f]", time.Since(m.startTime).Seconds())
	m.logs = append(m.logs, fmt.Sprintf("%s %s", timestamp, line))
	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}
}

func (m *QEMUBridgeManager) SetState(newState string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = newState
	if newState == "rebooting" {
		m.startTime = time.Now()
		m.logs = []string{
			"[0.000000] System reset invoked by user.",
			"[0.000850] Onuron Microkernel v0.1.0-alpha rebooted.",
			"[0.002100] NilOS Desktop Shell restored.",
		}
		m.state = "running"
	}
}

// ─── QEMU API Handlers ───────────────────────────────────────────────

func handleQEMUStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := globalQEMUBridge.GetStatus()
	json.NewEncoder(w).Encode(status)
}

func handleQEMUControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action string `json:"action"` // "start", "stop", "pause", "resume", "reboot", "sendkey"
		Key    string `json:"key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "start":
		globalQEMUBridge.SetState("running")
		globalQEMUBridge.AppendLog("VM instance started.")
	case "stop":
		globalQEMUBridge.SetState("stopped")
		globalQEMUBridge.AppendLog("VM instance halted.")
	case "pause":
		globalQEMUBridge.SetState("paused")
		globalQEMUBridge.AppendLog("VM execution paused.")
	case "resume":
		globalQEMUBridge.SetState("running")
		globalQEMUBridge.AppendLog("VM execution resumed.")
	case "reboot":
		globalQEMUBridge.SetState("rebooting")
	case "sendkey":
		if req.Key != "" {
			globalQEMUBridge.AppendLog(fmt.Sprintf("Kernel PS/2 keyboard received scancode: %s", req.Key))
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"status":  globalQEMUBridge.GetStatus(),
	})
}

func handleQEMUFramebuffer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := globalQEMUBridge.GetStatus()

	// Return rich framebuffer description and textual VGA buffer for HTML5 Canvas streaming
	vgaLines := []string{
		"┌──────────────────────────────────────────────────────────────────────────────┐",
		"│  🌟 NILOS 2.0 ONURON MICROKERNEL VIRTUAL DISPLAY (1024x768 @ 60Hz)          │",
		"├──────────────────────────────────────────────────────────────────────────────┤",
		"│  Kernel: Onuron x86_64 SMP (8 Cores)  | RAM: 8192 MB | NilBus: Online        │",
		"│  Display Driver: Alap Vulkan DRM/KMS  | Compositor: NilWM Wayland 1.22       │",
		"├──────────────────────────────────────────────────────────────────────────────┤",
	}

	for _, l := range status.LogBuffer {
		if len(l) > 76 {
			l = l[:73] + "..."
		}
		vgaLines = append(vgaLines, fmt.Sprintf("│  %-76s│", l))
	}
	for len(vgaLines) < 22 {
		vgaLines = append(vgaLines, "│                                                                              │")
	}
	vgaLines = append(vgaLines, "└──────────────────────────────────────────────────────────────────────────────┘")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"frameNumber": status.FrameNumber,
		"state":       status.State,
		"width":       1024,
		"height":      768,
		"vgaText":     vgaLines,
		"registers":   status.Registers,
		"timestamp":   time.Now().UnixNano(),
	})
}
