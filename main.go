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
)

var (
	store     *Store
	startTime = time.Now()
)

func main() {
	var err error
	store, err = NewStore("notes.json")
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	port := 8085

	mux := http.NewServeMux()

	// ─── 1. Notes API Routes ──────────────────────────────────────────
	mux.HandleFunc("/api/notes", handleNotes)
	mux.HandleFunc("/api/notes/", handleSingleNote)
	mux.HandleFunc("/api/analytics", handleAnalytics)

	// ─── 2. Emulator Subsystem API Routes ──────────────────────────────
	mux.HandleFunc("/api/emulator/status", handleEmulatorStatus)
	mux.HandleFunc("/api/emulator/packages", handleEmulatorPackages)
	mux.HandleFunc("/api/emulator/terminal", handleEmulatorTerminal)
	mux.HandleFunc("/api/pos/products", handlePOSProducts)
	mux.HandleFunc("/api/pos/checkout", handlePOSCheckout)
	mux.HandleFunc("/api/music/tracks", handleMusicTracks)
	mux.HandleFunc("/api/files/browse", handleFilesBrowse)

	// ─── 3. QEMU Kernel Bridge & Framebuffer Routes ───────────────────
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

	// ─── 4. Nil-LSP (Language Server Protocol) Routes ──────────────────
	mux.HandleFunc("/api/lsp/analyze", handleLSPAnalyze)
	mux.HandleFunc("/api/lsp/complete", handleLSPComplete)
	mux.HandleFunc("/api/lsp/hover", handleLSPHover)

	// ─── 5. NDB (Nil Debugger) Remote Protocol Routes ──────────────────
	mux.HandleFunc("/api/ndb/session", handleNDBSession)
	mux.HandleFunc("/api/ndb/step", handleNDBStep)
	mux.HandleFunc("/api/ndb/breakpoints", handleNDBBreakpoints)
	mux.HandleFunc("/api/ndb/eval", handleNDBEval)

	// ─── 6. IDE Extensions (VSCode & Android Studio) ──────────────────
	mux.HandleFunc("/api/extensions/list", handleExtensionsList)
	mux.HandleFunc("/api/extensions/vscode/download", handleVSCodeDownload)
	mux.HandleFunc("/api/extensions/android-studio/download", handleAndroidStudioDownload)

	// ─── 4. Static Assets ─────────────────────────────────────────────
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/", fs)

	fmt.Println("==============================================================")
	fmt.Println("  📱 নীল ওএস ডিভাইস এমুলেটর স্টুডিও — NILOS STUDIO EMULATOR")
	fmt.Printf("  🌐 লোকাল সার্ভার চালু হয়েছে: http://localhost:%d\n", port)
	fmt.Printf("  📱 এমুলেটর ডিভাইস ইন্টারফেস: http://localhost:%d/device.html\n", port)
	fmt.Println("  🖥️  QEMU x86_64 ও ভার্চুয়াল হার্ডওয়্যার: সক্রিয় (Ready)")
	fmt.Println("==============================================================")

	// Auto-open browser in non-headless run
	if len(os.Args) > 1 && os.Args[1] == "--open" {
		go openBrowser(fmt.Sprintf("http://localhost:%d/device.html", port))
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
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

	// Sub-actions: /api/notes/:id/pin, /api/notes/:id/favorite, /api/notes/:id/toggle-checklist
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
		words := len(strings.Fields(n.Content))
		totalWords += words
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

	analytics := map[string]interface{}{
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
	}
	json.NewEncoder(w).Encode(analytics)
}

// ─── Emulator API Handlers ───────────────────────────────────────────
func handleEmulatorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	notes := store.GetAll()
	activeNotes := 0
	for _, n := range notes {
		if !n.IsTrashed {
			activeNotes++
		}
	}

	uptimeSeconds := int(time.Since(startTime).Seconds())

	status := map[string]interface{}{
		"os": map[string]interface{}{
			"name":         "NilOS (নীল ওএস)",
			"version":      "2.0.0-pro",
			"kernel":       "Onuron Microkernel v0.1.0-alpha",
			"arch":         "x86_64 / arm64-hybrid",
			"displayMode":  "Wayland 1.22 + Vulkan UI Compositor",
			"nilbusStatus": "Online (12 active channels)",
			"uptime":       fmt.Sprintf("%d seconds", uptimeSeconds),
		},
		"hardware": map[string]interface{}{
			"cpu":            "Onuron NilCore x86_64 (Octa-Core @ 3.4GHz)",
			"cpuLoadPct":     14.5,
			"ramTotalMB":     8192,
			"ramUsedMB":      1420,
			"display":        `6.7" Super Retina AMOLED (120Hz Touch)`,
			"thermalPrinter": "NilThermal 58mm Paper ESC/POS — Connected",
			"scanner":        "EAN-13 / QR Laser Engine — Ready",
			"battery": map[string]interface{}{
				"level":     98,
				"charging":  true,
				"healthPct": 100,
			},
			"network": map[string]interface{}{
				"carrier": "Onuron Net 5G",
				"signal":  "Excellent (-65 dBm)",
				"speed":   "1.2 Gbps",
				"online":  true,
			},
		},
		"apps": []map[string]interface{}{
			{"id": "notes", "name": "নীলাং নোটবুক", "pkg": "org.nilang.notes", "version": "1.0.0", "state": "running", "notesCount": activeNotes},
			{"id": "calc", "name": "ক্যালকুলেটর", "pkg": "org.nilang.calculator", "version": "1.0.0", "state": "ready"},
			{"id": "pos", "name": "পিওএস টার্মিনাল", "pkg": "org.nilang.pos", "version": "1.0.0", "state": "ready"},
			{"id": "music", "name": "মিউজিক প্লেয়ার", "pkg": "org.nilang.music", "version": "1.0.0", "state": "ready"},
			{"id": "shell", "name": "নীলাং টার্মিনাল", "pkg": "org.nilang.shell", "version": "1.0.0", "state": "ready"},
			{"id": "store", "name": "প্যাকেজ স্টোর", "pkg": "org.nilang.pkgmgr", "version": "1.0.0", "state": "ready"},
			{"id": "files", "name": "ফাইল ম্যানেজার", "pkg": "org.nilang.nilfs", "version": "1.0.0", "state": "ready"},
			{"id": "settings", "name": "সেটিংস", "pkg": "org.nilos.settings", "version": "2.0.0", "state": "ready"},
		},
	}

	json.NewEncoder(w).Encode(status)
}

func handleEmulatorPackages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pkgs := []map[string]interface{}{
		{
			"id":          "notes-app",
			"name":        "নীলাং নোটবুক (Nilang Notes)",
			"version":     "1.0.0",
			"bundle":      "notes-app-nilLang-1.0.0.nilax",
			"size":        "40.3 KB",
			"icon":        "📝",
			"author":      "Joy Sriram Sarkar",
			"permissions": []string{"Filesystem", "Database", "Network"},
			"signed":      true,
			"signature":   "Ed25519 (Valid: Joyshriram Sarkar)",
			"installed":   true,
			"description": "সম্পূর্ণ নীলাং ও আলাপ ফ্রেমওয়ার্কে তৈরি রিচ নোটবুক অ্যাপ।",
		},
		{
			"id":          "pos-app",
			"name":        "অনুরণ রিটেইল পিওএস (Retail POS)",
			"version":     "1.0.0",
			"bundle":      "pos-app-1.0.0.nilax",
			"size":        "21.4 KB",
			"icon":        "🏪",
			"author":      "POS Developer",
			"permissions": []string{"Filesystem", "Database", "Printer", "Scanner"},
			"signed":      true,
			"signature":   "Ed25519 (Valid: POS Developer)",
			"installed":   true,
			"description": "বারকোড সার্চ, ইনভেন্টরি, কার্ট, ভ্যাট ও থার্মাল রিসিট প্রিন্টার সহ পূর্ণাঙ্গ রিটেইল পয়েন্ট-অব-সেল।",
		},
		{
			"id":          "calculator-app",
			"name":        "মডার্ন ক্যালকুলেটর (Calculator)",
			"version":     "1.0.0",
			"bundle":      "calculator-1.0.0.nilax",
			"size":        "18.2 KB",
			"icon":        "🧮",
			"author":      "Nilang Core Team",
			"permissions": []string{"AlapUI"},
			"signed":      true,
			"signature":   "Ed25519 (Valid: System)",
			"installed":   true,
			"description": "বেসিক ও সায়েন্টিফিক মোড, হিস্ট্রি রিল ও মেমোরি ফাংশনযুক্ত ক্যালকুলেটর।",
		},
		{
			"id":          "music-app",
			"name":        "নীলাং মিউজিক স্ট্রিমার (Music Player)",
			"version":     "1.0.0",
			"bundle":      "music-streaming-1.0.0.nilax",
			"size":        "24.8 KB",
			"icon":        "🎵",
			"author":      "Nilang Media Lab",
			"permissions": []string{"Audio", "Network", "Filesystem"},
			"signed":      true,
			"signature":   "Ed25519 (Valid: Nilang Core)",
			"installed":   true,
			"description": "বাংলা গান ও অডিও স্ট্রিমিং প্লেয়ার উইথ লাইভ সাউন্ড ওয়েভফর্ম।",
		},
	}

	json.NewEncoder(w).Encode(pkgs)
}

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

	nilBin := filepath.Join("..", "nilLang", "bin", "nil.exe")
	if _, err := os.Stat(nilBin); os.IsNotExist(err) {
		nilBin = "nil"
	}

	// 1. Command execution (e.g. "nil --version", "nil run ...")
	if req.Command != "" {
		parts := strings.Fields(req.Command)
		if len(parts) == 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "output": ""})
			return
		}

		bin := parts[0]
		args := parts[1:]
		if bin == "nil" {
			bin = nilBin
		}

		cmd := exec.Command(bin, args...)
		out, err := cmd.CombinedOutput()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  err == nil,
			"output":   string(out),
			"exitCode": cmd.ProcessState.ExitCode(),
		})
		return
	}

	// 2. Direct code evaluation
	if req.Code != "" {
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nil_eval_%d.nil", time.Now().UnixNano()))
		if err := os.WriteFile(tmpFile, []byte(req.Code), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmpFile)

		cmd := exec.Command(nilBin, "run", tmpFile)
		out, err := cmd.CombinedOutput()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"output":  string(out),
			"error":   fmt.Sprintf("%v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  "Nilang v1.0.0 Interactive REPL Ready. Type code or command.",
	})
}

func handlePOSProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	products := GetDefaultPOSCatalog()
	json.NewEncoder(w).Encode(products)
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
	vat := (subtotal - discount) * 0.05 // 5% VAT
	total := (subtotal - discount) + vat

	now := time.Now()
	invoiceNo := fmt.Sprintf("INV-%s-%03d", now.Format("20060102"), now.Unix()%1000)

	custName := req.CustomerName
	if custName == "" {
		custName = "সাধারণ ক্রেতা (Walk-in Customer)"
	}

	receipt := POSReceipt{
		InvoiceNo:     invoiceNo,
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
	tracks := GetDefaultMusicTracks()
	json.NewEncoder(w).Encode(tracks)
}

func handleFilesBrowse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" {
		reqPath = "."
	}

	entries, err := os.ReadDir(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var fileList []map[string]interface{}
	for _, e := range entries {
		info, err := e.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		fileList = append(fileList, map[string]interface{}{
			"name":  e.Name(),
			"isDir": e.IsDir(),
			"size":  size,
			"path":  filepath.Join(reqPath, e.Name()),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"currentPath": reqPath,
		"entries":     fileList,
	})
}

// ─── QEMU & Nilang Legacy Handlers ────────────────────────────────────
func handleLaunchQEMU(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	qemuPath := `C:\Program Files\qemu\qemu-system-x86_64.exe`
	bootImg := filepath.Join("..", "notes-app-nilLang", "qemu", "notes-os.img")

	if _, err := os.Stat(qemuPath); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "QEMU executable not found at: " + qemuPath,
		})
		return
	}

	cmd := exec.Command(qemuPath, "-name", "NilOS Studio - Onuron Virtual OS", "-drive", fmt.Sprintf("format=raw,file=%s", bootImg), "-m", "512M", "-vga", "std", "-boot", "a")
	if err := cmd.Start(); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "QEMU Virtual Machine launched successfully!",
		"pid":     cmd.Process.Pid,
	})
}

func handleRunNilang(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nilBin := filepath.Join("..", "nilLang", "bin", "nil.exe")
	if _, err := os.Stat(nilBin); os.IsNotExist(err) {
		nilBin = "nil"
	}

	srcFile := filepath.Join("..", "notes-app-nilLang", "src", "main.nil")
	cmd := exec.Command(nilBin, "run", srcFile)
	cmd.Stdin = strings.NewReader("v\nq\n")
	out, err := cmd.CombinedOutput()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"output":  string(out),
		"error":   fmt.Sprintf("%v", err),
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
