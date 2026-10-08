package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/api"
	"github.com/joysriramsarkar/nilos-emulator/internal/config"
	"github.com/joysriramsarkar/nilos-emulator/internal/emulator"
	"github.com/joysriramsarkar/nilos-emulator/internal/sandbox"
	"github.com/joysriramsarkar/nilos-emulator/internal/supervisor"
)

var (
	store     *Store
	startTime = time.Now()

	cfg      config.Config
	devMgr   *emulator.Manager
	sup      *supervisor.Supervisor
	workspace *sandbox.Guard
)

func main() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	store, err = NewStore(filepath.Join(cfg.DataDir, "notes.json"))
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	devMgr, err = emulator.NewManager(cfg.DataDir)
	if err != nil {
		log.Fatalf("device manager: %v", err)
	}
	sup = supervisor.New(devMgr, cfg)
	devMgr.SetLauncher(sup)

	// The workspace guard restricts legacy file browsing/editing endpoints to
	// the current directory and the emulator data directory.
	cwd, _ := os.Getwd()
	workspace = sandbox.NewGuard(cwd, cfg.DataDir)

	mux := http.NewServeMux()

	// ─── 1. Notes API Routes ──────────────────────────────────────────
	mux.HandleFunc("/api/notes", handleNotes)
	mux.HandleFunc("/api/notes/", handleSingleNote)
	mux.HandleFunc("/api/analytics", handleAnalytics)

	// ─── 2. Emulator Subsystem API Routes (legacy, hardened) ───────────
	mux.HandleFunc("/api/emulator/status", handleEmulatorStatus)
	mux.HandleFunc("/api/emulator/packages", handleEmulatorPackages)
	mux.HandleFunc("/api/emulator/terminal", handleEmulatorTerminal)
	mux.HandleFunc("/api/pos/products", handlePOSProducts)
	mux.HandleFunc("/api/pos/checkout", handlePOSCheckout)
	mux.HandleFunc("/api/music/tracks", handleMusicTracks)
	mux.HandleFunc("/api/files/browse", handleFilesBrowse)

	// ─── 3. QEMU Kernel Bridge (honest status, optional system mode) ───
	mux.HandleFunc("/api/qemu/launch", handleLaunchQEMU)
	mux.HandleFunc("/api/qemu/status", handleQEMUStatus)
	mux.HandleFunc("/api/qemu/control", handleQEMUControl)
	mux.HandleFunc("/api/qemu/framebuffer", handleQEMUFramebuffer)

	// ─── 4. NilLang Real App Execution & Compiler Bridge ───────────────
	mux.HandleFunc("/api/nilang/apps", handleNilangApps)
	mux.HandleFunc("/api/nilang/file/read", handleNilangReadFile)
	mux.HandleFunc("/api/nilang/file/save", handleNilangSaveFile)
	mux.HandleFunc("/api/nilang/execute", handleNilangExecute)
	mux.HandleFunc("/api/nilang/hir", handleNilangHIR)
	mux.HandleFunc("/api/nilang/run", handleRunNilang)

	// ─── 5. Nil-LSP / NDB Debugger ─────────────────────────────────────
	mux.HandleFunc("/api/lsp/analyze", handleLSPAnalyze)
	mux.HandleFunc("/api/lsp/complete", handleLSPComplete)
	mux.HandleFunc("/api/lsp/hover", handleLSPHover)
	mux.HandleFunc("/api/ndb/session", handleNDBSession)
	mux.HandleFunc("/api/ndb/step", handleNDBStep)
	mux.HandleFunc("/api/ndb/breakpoints", handleNDBBreakpoints)
	mux.HandleFunc("/api/ndb/eval", handleNDBEval)

	// ─── 6. IDE Extensions ─────────────────────────────────────────────
	mux.HandleFunc("/api/extensions/list", handleExtensionsList)
	mux.HandleFunc("/api/extensions/vscode/download", handleVSCodeDownload)
	mux.HandleFunc("/api/extensions/android-studio/download", handleAndroidStudioDownload)

	// ─── 7. New NilStudio control API (devices, apps, HAL, events) ─────
	apiSrv := api.New(cfg, devMgr, sup)
	mux.Handle("/api/v1/", apiSrv.Handler())

	// ─── 8. Static Studio frontend ─────────────────────────────────────
	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := cfg.LoopbackAddr()
	fmt.Println("==============================================================")
	fmt.Println("  📱 NILOS STUDIO EMULATOR — Nilang App Runtime Emulator")
	fmt.Printf("  🌐 Control API + Studio:  http://%s/\n", addr)
	fmt.Printf("  📱 Device UI:             http://%s/device.html\n", addr)
	fmt.Printf("  🧩 Studio shell:          http://%s/studio.html\n", addr)
	fmt.Printf("  💾 Data dir:              %s\n", cfg.DataDir)
	if cfg.AuthToken == "" {
		fmt.Println("  🔓 Auth token disabled (loopback-only). Set NILEMU_TOKEN to enable.")
	}
	fmt.Println("==============================================================")

	if !cfg.Headless && len(os.Args) > 1 && os.Args[1] == "--open" {
		go openBrowser(fmt.Sprintf("http://%s/studio.html", addr))
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ─── Notes Handlers ──────────────────────────────────────────────────

func handleNotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		notes := store.GetAll()
		json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var n Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		created, err := store.Create(n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSingleNote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	parts := strings.Split(path, "/")
	id := parts[0]

	if id == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	if len(parts) > 1 {
		action := parts[1]
		note, found := store.Get(id)
		if !found {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}

		switch action {
		case "pin":
			note.IsPinned = !note.IsPinned
			updated, err := store.Update(id, note)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(updated)
			return

		case "favorite":
			note.IsFavorite = !note.IsFavorite
			updated, err := store.Update(id, note)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(updated)
			return

		case "toggle-checklist":
			if len(parts) > 2 {
				checkID := parts[2]
				for i, c := range note.Checklist {
					if c.ID == checkID {
						note.Checklist[i].Completed = !note.Checklist[i].Completed
						break
					}
				}
				updated, err := store.Update(id, note)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(updated)
				return
			}
		}
	}

	switch r.Method {
	case http.MethodGet:
		note, found := store.Get(id)
		if !found {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(note)

	case http.MethodPut:
		var n Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := store.Update(id, n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(updated)

	case http.MethodDelete:
		if err := store.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": id})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	notes := store.GetAll()
	totalNotes := len(notes)
	activeNotes := 0
	pinnedNotes := 0
	favoriteNotes := 0
	archivedNotes := 0
	trashedNotes := 0
	totalWords := 0
	totalChars := 0
	totalTodos := 0
	completedTodos := 0

	for _, n := range notes {
		if n.IsTrashed {
			trashedNotes++
			continue
		}
		activeNotes++
		if n.IsPinned {
			pinnedNotes++
		}
		if n.IsFavorite {
			favoriteNotes++
		}
		if n.IsArchived {
			archivedNotes++
		}
		totalWords += len(strings.Fields(n.Content))
		totalChars += len(n.Content)
		for _, c := range n.Checklist {
			totalTodos++
			if c.Completed {
				completedTodos++
			}
		}
	}

	todoProgress := 0
	if totalTodos > 0 {
		todoProgress = int(float64(completedTodos) / float64(totalTodos) * 100)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalNotes":     totalNotes,
		"activeNotes":    activeNotes,
		"pinnedNotes":    pinnedNotes,
		"favoriteNotes":  favoriteNotes,
		"archivedNotes":  archivedNotes,
		"trashedNotes":   trashedNotes,
		"totalWords":     totalWords,
		"totalChars":     totalChars,
		"totalTodos":     totalTodos,
		"completedTodos": completedTodos,
		"todoProgress":   todoProgress,
	})
}

// ─── Emulator API Handlers (now backed by real device manager) ────────

func handleEmulatorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	devices := devMgr.ListDevices()
	sessions := sup.Sessions()
	status := map[string]interface{}{
		"os": map[string]interface{}{
			"name":        "NilOS Lightweight Runtime",
			"version":     "0.2.0",
			"runtime":     "NilHost Protocol v1",
			"mode":        "app-runtime",
			"uptime":      fmt.Sprintf("%d seconds", int(time.Since(startTime).Seconds())),
			"devices":     len(devices),
			"runningApps": len(sessions),
		},
		"deviceManager": map[string]interface{}{
			"devices":  devices,
			"sessions": sessions,
		},
	}
	json.NewEncoder(w).Encode(status)
}

func handleEmulatorPackages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// List installed apps across devices as the package inventory.
	type pkg struct {
		Device      string `json:"device"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Signed      bool   `json:"signed"`
		Permissions int    `json:"permissions"`
	}
	var out []pkg
	for _, d := range devMgr.ListDevices() {
		apps, _ := devMgr.ListInstalled(d.ID)
		for _, a := range apps {
			out = append(out, pkg{Device: d.ID, ID: a.ID, Name: a.Name, Version: a.Version, Signed: a.Signed, Permissions: len(a.Permissions)})
		}
	}
	if out == nil {
		out = []pkg{}
	}
	json.NewEncoder(w).Encode(out)
}

// handleEmulatorTerminal no longer executes arbitrary host commands. It runs a
// small virtual shell operating on the emulator's own state.
func handleEmulatorTerminal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Code    string `json:"code"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out := runVirtualShell(req.Command)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"output":   out,
		"exitCode": 0,
	})
}

func runVirtualShell(cmd string) string {
	fields := strings.Fields(strings.TrimSpace(cmd))
	if len(fields) == 0 {
		return "NilOS virtual shell. Try: help, devices, apps, version"
	}
	switch fields[0] {
	case "help":
		return "commands: help, version, devices, apps, sessions, tree <app>"
	case "version":
		return "nilemu 0.2.0 · NilHost protocol v1 · built " + runtime.Version()
	case "devices":
		var b strings.Builder
		for _, d := range devMgr.ListDevices() {
			fmt.Fprintf(&b, "%s  %s  %s  %s\n", d.ID, d.State, d.ProfileID, d.Name)
		}
		return b.String()
	case "apps":
		var b strings.Builder
		for _, d := range devMgr.ListDevices() {
			apps, _ := devMgr.ListInstalled(d.ID)
			for _, a := range apps {
				fmt.Fprintf(&b, "%s  %s  %s\n", d.ID, a.ID, a.Version)
			}
		}
		return b.String()
	case "sessions":
		var b strings.Builder
		for _, s := range sup.Sessions() {
			fmt.Fprintf(&b, "%s/%s  %s  frames=%d patches=%d\n", s.DeviceID, s.AppID, s.State, s.Frames, s.Patches)
		}
		return b.String()
	default:
		return fmt.Sprintf("unknown command %q (this is a virtual shell, host commands are disabled)", fields[0])
	}
}

func handlePOSProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetDefaultPOSCatalog())
}

func handlePOSCheckout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req POSCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var subtotal float64
	for _, item := range req.Items {
		subtotal += item.Product.Price * float64(item.Quantity)
	}
	discount := subtotal * (req.DiscountPct / 100.0)
	vat := (subtotal - discount) * 0.05
	total := (subtotal - discount) + vat

	now := time.Now()
	custName := req.CustomerName
	if custName == "" {
		custName = "সাধারণ ক্রেতা (Walk-in Customer)"
	}
	receipt := POSReceipt{
		InvoiceNo:     fmt.Sprintf("INV-%s-%03d", now.Format("20060102"), now.Unix()%1000),
		Date:          now.Format("02/01/2006"),
		Time:          now.Format("03:04:05 PM"),
		CustomerName:  custName,
		CustomerPhone: req.CustomerPhone,
		Items:         req.Items,
		Subtotal:      subtotal,
		Discount:      discount,
		VAT:           vat,
		Total:         total,
		PaymentMethod: req.PaymentMethod,
		PrintedAt:     now.Unix(),
		Cashier:       "নিলয় চৌধুরী (POS-01)",
	}
	json.NewEncoder(w).Encode(receipt)
}

func handleMusicTracks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetDefaultMusicTracks())
}

// handleFilesBrowse is restricted to the workspace (cwd + data dir).
func handleFilesBrowse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" {
		reqPath = "."
	}
	safe, err := workspace.Check(reqPath)
	if err != nil {
		http.Error(w, "path outside workspace", http.StatusForbidden)
		return
	}
	entries, err := os.ReadDir(safe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var fileList []map[string]interface{}
	for _, e := range entries {
		info, ierr := e.Info()
		size := int64(0)
		if ierr == nil {
			size = info.Size()
		}
		fileList = append(fileList, map[string]interface{}{
			"name":  e.Name(),
			"isDir": e.IsDir(),
			"size":  size,
			"path":  filepath.Join(safe, e.Name()),
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"currentPath": safe,
		"entries":     fileList,
	})
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}
