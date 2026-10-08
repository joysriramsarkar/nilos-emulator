package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// NilAppMetadata describes a real NilLang application found in the ecosystem
type NilAppMetadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	MainNilFile string   `json:"mainNilFile"`
	Icon        string   `json:"icon"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Category    string   `json:"category"`
	Files       []string `json:"files"`
}

type NilExecutionResult struct {
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exitCode"`
	HIR       string `json:"hir,omitempty"`
	Execution string `json:"executionTime"`
}

type NilRunnerManager struct {
	mu           sync.RWMutex
	nilBinary    string
	workspaceDir string
	activeAppID  string
	runningCmd   *exec.Cmd
	appProcesses map[string]*exec.Cmd
}

var globalNilRunner = NewNilRunnerManager()

func NewNilRunnerManager() *NilRunnerManager {
	// Look for nil.exe
	candidates := []string{
		filepath.Join("..", "nilLang", "bin", "nil.exe"),
		filepath.Join("..", "nilLang", "nil-test.exe"),
		filepath.Join(".", "bin", "nil.exe"),
		"nil.exe",
		"nil",
	}

	nilBin := "nil"
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			nilBin = c
			break
		}
	}

	return &NilRunnerManager{
		nilBinary:    nilBin,
		workspaceDir: filepath.Join(".."),
		appProcesses: make(map[string]*exec.Cmd),
	}
}

// GetDiscoveredApps discovers real NilLang applications in parent directory
func (m *NilRunnerManager) GetDiscoveredApps() []NilAppMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var apps []NilAppMetadata

	// 1. Notes App
	notesPath := filepath.Join("..", "notes-app-nilLang")
	notesMain := filepath.Join(notesPath, "src", "main.nil")
	if _, err := os.Stat(notesMain); err == nil {
		apps = append(apps, NilAppMetadata{
			ID:          "notes-app",
			Name:        "নীলাং নোটবুক (Nilang Notes)",
			Description: "সম্পূর্ণ নীলাং ও আলাপ ডিক্লারেটিভ ফ্রেমে তৈরি আসল প্রোডাকশন নোটবুক অ্যাপ।",
			Path:        notesPath,
			MainNilFile: notesMain,
			Icon:        "📝",
			Version:     "1.0.0",
			Author:      "Joy Sriram Sarkar",
			Category:    "Productivity",
			Files:       m.scanNilFiles(notesPath),
		})
	}

	// 2. Retail POS App
	posPath := filepath.Join("..", "pos-app-nilLang")
	posMain := filepath.Join(posPath, "src", "main.nil")
	if _, err := os.Stat(posPath); err == nil {
		apps = append(apps, NilAppMetadata{
			ID:          "pos-app",
			Name:        "অনুরণ রিটেইল পিওএস (Retail POS)",
			Description: "বারকোড স্ক্যানার, ইনভেন্টরি, ৫% ভ্যাট ও থার্মাল পেপার প্রিন্টার সহ পয়েন্ট-অব-সেল।",
			Path:        posPath,
			MainNilFile: posMain,
			Icon:        "🏪",
			Version:     "1.0.0",
			Author:      "Nilang Core Team",
			Category:    "Commerce",
			Files:       m.scanNilFiles(posPath),
		})
	}

	// 3. Calculator App
	calcPath := filepath.Join("..", "calculator-nilLang")
	calcMain := filepath.Join(calcPath, "src", "main.nil")
	if _, err := os.Stat(calcPath); err == nil {
		apps = append(apps, NilAppMetadata{
			ID:          "calc-app",
			Name:        "মডার্ন ক্যালকুলেটর (Nilang Calculator)",
			Description: "বেসিক ও সায়েন্টিফিক মোড, হিস্ট্রি মেমোরি সহ ক্যালকুলেটর অ্যাপ্লিকেশন।",
			Path:        calcPath,
			MainNilFile: calcMain,
			Icon:        "🧮",
			Version:     "1.0.0",
			Author:      "Nilang Team",
			Category:    "Utility",
			Files:       m.scanNilFiles(calcPath),
		})
	}

	// 4. Music Streaming App
	musicPath := filepath.Join("..", "music-streaming-nilLang")
	musicMain := filepath.Join(musicPath, "src", "main.nil")
	if _, err := os.Stat(musicPath); err == nil {
		apps = append(apps, NilAppMetadata{
			ID:          "music-app",
			Name:        "নীলাং মিউজিক স্ট্রিমার (Music Streamer)",
			Description: "বাংলা গান প্লেলিস্ট ও রিয়েল-টাইম ক্যানভাস সাউন্ড ইকুয়ালাইজার।",
			Path:        musicPath,
			MainNilFile: musicMain,
			Icon:        "🎵",
			Version:     "1.0.0",
			Author:      "Media Lab",
			Category:    "Entertainment",
			Files:       m.scanNilFiles(musicPath),
		})
	}

	// 5. Game App
	gamePath := filepath.Join("..", "nilLang", "my-nilang-game")
	if _, err := os.Stat(gamePath); err == nil {
		apps = append(apps, NilAppMetadata{
			ID:          "game-app",
			Name:        "নীলাং গেম স্টুডিও (Nilang Game)",
			Description: "2D ক্যানভাস ফিজিক্স ও জেসচার কন্ট্রোল গেম।",
			Path:        gamePath,
			MainNilFile: filepath.Join(gamePath, "main.nil"),
			Icon:        "🎮",
			Version:     "1.0.0",
			Author:      "Game Lab",
			Category:    "Games",
			Files:       m.scanNilFiles(gamePath),
		})
	}

	return apps
}

func (m *NilRunnerManager) scanNilFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && (strings.HasSuffix(path, ".nil") || strings.HasSuffix(path, ".json")) {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, rel)
		}
		return nil
	})
	return files
}

// ExecuteNilCode runs code or a file with nil.exe
func (m *NilRunnerManager) ExecuteNilCode(code string, filePath string, input string) NilExecutionResult {
	start := time.Now()

	var targetFile string
	var cleanup bool

	if filePath != "" && fileExists(filePath) {
		targetFile = filePath
	} else if code != "" {
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nilos_eval_%d.nil", time.Now().UnixNano()))
		if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
			return NilExecutionResult{
				Success:  false,
				Error:    fmt.Sprintf("Failed to write temporary .nil file: %v", err),
				ExitCode: 1,
			}
		}
		targetFile = tmpFile
		cleanup = true
	} else {
		return NilExecutionResult{
			Success:  false,
			Error:    "No code or file path provided to execute",
			ExitCode: 1,
		}
	}

	if cleanup {
		defer os.Remove(targetFile)
	}

	// 1. Execute program with nil.exe run
	cmd := exec.Command(m.nilBinary, "run", targetFile)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	} else {
		// Send non-blocking quit or initial command
		cmd.Stdin = strings.NewReader("v\nq\n")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	outStr := stdout.String()
	if stderr.Len() > 0 {
		if outStr != "" {
			outStr += "\n"
		}
		outStr += stderr.String()
	}

	// 2. Capture HIR (High-Level Intermediate Representation) from nil.exe hir
	var hirOutput string
	hirCmd := exec.Command(m.nilBinary, "hir", targetFile)
	if hirBytes, hirErr := hirCmd.CombinedOutput(); hirErr == nil {
		hirOutput = string(hirBytes)
	}

	duration := time.Since(start).String()

	return NilExecutionResult{
		Success:   err == nil,
		Output:    outStr,
		Error:     fmt.Sprintf("%v", err),
		ExitCode:  exitCode,
		HIR:       hirOutput,
		Execution: duration,
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// ─── Nil Runner API Handlers ─────────────────────────────────────────

func handleNilangApps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	apps := globalNilRunner.GetDiscoveredApps()
	json.NewEncoder(w).Encode(apps)
}

func handleNilangReadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	safe, err := workspace.Check(filePath)
	if err != nil {
		http.Error(w, "path outside workspace", http.StatusForbidden)
		return
	}
	content, err := os.ReadFile(safe)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    safe,
		"content": string(content),
		"size":    len(content),
	})
}

func handleNilangSaveFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	safe, err := workspace.Check(req.Path)
	if err != nil {
		http.Error(w, "path outside workspace", http.StatusForbidden)
		return
	}
	if err := os.MkdirAll(filepath.Dir(safe), 0o755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(safe, []byte(req.Content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    safe,
		"message": "File saved successfully for hot reload.",
	})
}

func handleNilangExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code     string `json:"code"`
		FilePath string `json:"filePath"`
		Input    string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := globalNilRunner.ExecuteNilCode(req.Code, req.FilePath, req.Input)
	json.NewEncoder(w).Encode(result)
}

func handleNilangHIR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	filePath := r.URL.Query().Get("path")

	var cmd *exec.Cmd
	if filePath != "" && fileExists(filePath) {
		cmd = exec.Command(globalNilRunner.nilBinary, "hir", filePath)
	} else {
		// Default to notes main.nil
		notesMain := filepath.Join("..", "notes-app-nilLang", "src", "main.nil")
		cmd = exec.Command(globalNilRunner.nilBinary, "hir", notesMain)
	}

	out, err := cmd.CombinedOutput()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"hir":     string(out),
		"path":    filePath,
	})
}

// handleRunNilang is the legacy batch-run endpoint. Prefer the live session
// API (/api/v1) for interactive apps.
func handleRunNilang(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Code     string `json:"code"`
		FilePath string `json:"filePath"`
		Input    string `json:"input"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	result := globalNilRunner.ExecuteNilCode(req.Code, req.FilePath, req.Input)
	json.NewEncoder(w).Encode(result)
}
