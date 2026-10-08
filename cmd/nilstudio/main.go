// Command nilstudio is a lightweight native shell for the emulator. It starts
// the nilemu control server (unless one is already running) and opens the
// Studio UI in the operating system's WebView in application mode — a
// frameless window without browser chrome.
//
// This is the "system WebView" approach the plan recommends (like Wails). It
// deliberately avoids bundling Chromium and needs no C toolchain; the server
// stays a single static Go binary.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8085", "server listen address")
	page := flag.String("page", "studio.html", "page to open")
	noOpen := flag.Bool("no-open", false, "start the server without opening a window")
	flag.Parse()

	base := "http://" + *addr
	url := base + "/" + *page

	serverProc, err := ensureServer(*addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nilstudio:", err)
		os.Exit(1)
	}
	if serverProc != nil {
		defer func() {
			_ = serverProc.Process.Kill()
		}()
	}

	if *noOpen {
		fmt.Println("nilstudio: server ready at", url, "(no window; press Ctrl+C to stop)")
		waitForSignal()
		return
	}

	if err := openAppWindow(url); err != nil {
		fmt.Fprintln(os.Stderr, "nilstudio: could not open a webview window:", err)
		fmt.Println("nilstudio: open this URL manually:", url)
	}
	waitForSignal()
}

// ensureServer starts a sibling `nilemu` binary if nothing is listening yet.
func ensureServer(addr string) (*exec.Cmd, error) {
	if healthy(addr) {
		return nil, nil
	}

	bin := siblingBinary("nilemu")
	if bin == "" {
		// Fall back to `go run .` when running from source.
		cmd := exec.Command("go", "run", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "NILEMU_PORT="+portOf(addr))
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start server: %w", err)
		}
		if err := waitHealthy(addr, 20*time.Second); err != nil {
			return nil, err
		}
		return cmd, nil
	}

	cmd := exec.Command(bin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NILEMU_PORT="+portOf(addr), "NILEMU_ADDR="+hostOf(addr))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	if err := waitHealthy(addr, 20*time.Second); err != nil {
		return nil, err
	}
	return cmd, nil
}

func healthy(addr string) bool {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get("http://" + addr + "/api/v1/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitHealthy(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if healthy(addr) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("server did not become healthy at %s", addr)
}

func siblingBinary(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	candidate := filepath.Join(dir, name+suffix)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// openAppWindow launches the OS webview in application mode.
func openAppWindow(url string) error {
	switch runtime.GOOS {
	case "windows":
		for _, browser := range []string{"msedge", "chrome"} {
			if path, err := exec.LookPath(browser); err == nil {
				return exec.Command(path,
					"--app="+url,
					"--window-size=1440,900",
					"--no-first-run",
					"--no-default-browser-check",
				).Start()
			}
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		// Safari/WebKit has no app mode; open as a normal window.
		return exec.Command("open", url).Start()
	default:
		for _, browser := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"} {
			if path, err := exec.LookPath(browser); err == nil {
				return exec.Command(path,
					"--app="+url,
					"--window-size=1440,900",
					"--no-first-run",
				).Start()
			}
		}
		return exec.Command("xdg-open", url).Start()
	}
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}

func portOf(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return "8085"
}

func hostOf(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
