// Package qemu talks to QEMU for the optional System Mode: it launches a VM
// with a QMP control channel and captures the real framebuffer via the
// `screendump` command. No fabricated registers or logs.
package qemu

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Client is a QEMU Machine Protocol (QMP) connection.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

// Dial connects to a QMP socket, consumes the greeting and negotiates
// capabilities.
func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, reader: bufio.NewReader(conn)}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	greeting, err := c.reader.ReadBytes('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("qmp greeting: %w", err)
	}
	if !bytes.Contains(greeting, []byte("QMP")) {
		conn.Close()
		return nil, fmt.Errorf("qmp: unexpected greeting %q", bytes.TrimSpace(greeting))
	}
	conn.SetReadDeadline(time.Time{})
	if _, err := c.Execute("qmp_capabilities", nil); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) readMessage() (map[string]any, error) {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if json.Unmarshal(line, &m) != nil {
			continue
		}
		// Skip asynchronous events; they have no "return"/"error".
		if _, ok := m["return"]; ok {
			return m, nil
		}
		if errVal, ok := m["error"]; ok {
			return nil, fmt.Errorf("qmp error: %v", errVal)
		}
		// event -> keep reading
	}
}

// Execute runs one QMP command.
func (c *Client) Execute(command string, args map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	req := map[string]any{"execute": command}
	if args != nil {
		req["arguments"] = args
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return nil, err
	}
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	m, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	ret, _ := m["return"].(map[string]any)
	return ret, nil
}

// Close closes the QMP connection.
func (c *Client) Close() error { return c.conn.Close() }

// LaunchOptions configures a QEMU instance.
type LaunchOptions struct {
	Binary   string
	Image    string
	MemoryMB int
	Accel    bool
	Extra    []string
}

// Instance is a running QEMU process with a QMP channel.
type Instance struct {
	cmd       *exec.Cmd
	qmpAddr   string
	client    *Client
	image     string
	startedAt time.Time
	mu        sync.Mutex
	lastErr   string
	exitCode  *int
}

// FindBinary locates qemu-system-x86_64.
func FindBinary() string {
	if v := os.Getenv("NILEMU_QEMU"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
		if p, err := exec.LookPath(v); err == nil {
			return p
		}
	}
	candidates := []string{"qemu-system-x86_64", "qemu-system-x86_64.exe", `C:\Program Files\qemu\qemu-system-x86_64.exe`}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// Launch starts a QEMU instance and waits for its QMP channel.
func Launch(opts LaunchOptions) (*Instance, error) {
	bin := opts.Binary
	if bin == "" {
		bin = FindBinary()
	}
	if bin == "" {
		return nil, fmt.Errorf("qemu-system-x86_64 was not found; install QEMU or set NILEMU_QEMU")
	}
	if opts.Image == "" {
		return nil, fmt.Errorf("no OS image configured (set NILEMU_OS_IMAGE)")
	}
	if _, err := os.Stat(opts.Image); err != nil {
		return nil, fmt.Errorf("OS image not found: %s", opts.Image)
	}
	if opts.MemoryMB <= 0 {
		opts.MemoryMB = 512
	}

	port, err := freePort()
	if err != nil {
		return nil, err
	}
	qmpAddr := fmt.Sprintf("127.0.0.1:%d", port)

	args := []string{
		"-m", fmt.Sprintf("%dM", opts.MemoryMB),
		"-display", "none",
		"-qmp", fmt.Sprintf("tcp:%s,server,nowait", qmpAddr),
		"-drive", fmt.Sprintf("format=raw,file=%s", opts.Image),
	}
	if opts.Accel {
		if flag := accelFlag(); flag != "" {
			args = append(args, "-accel", flag)
		}
	}
	args = append(args, opts.Extra...)

	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	inst := &Instance{cmd: cmd, qmpAddr: qmpAddr, image: opts.Image, startedAt: time.Now()}
	go inst.reap()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		client, err := Dial(qmpAddr)
		if err == nil {
			inst.client = client
			return inst, nil
		}
		inst.mu.Lock()
		done := inst.exitCode != nil
		errMsg := inst.lastErr
		inst.mu.Unlock()
		if done {
			return nil, fmt.Errorf("qemu exited early: %s", errMsg)
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("timed out waiting for QMP at %s", qmpAddr)
}

func (i *Instance) reap() {
	err := i.cmd.Wait()
	i.mu.Lock()
	defer i.mu.Unlock()
	code := 0
	if i.cmd.ProcessState != nil {
		code = i.cmd.ProcessState.ExitCode()
	}
	if err != nil {
		i.lastErr = err.Error()
	}
	i.exitCode = &code
}

// ScreenDump captures the real framebuffer as a PNG-ready image.
func (i *Instance) ScreenDump() ([]byte, error) {
	i.mu.Lock()
	client := i.client
	i.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("QMP not connected")
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("nilemu-fb-%d.ppm", time.Now().UnixNano()))
	defer os.Remove(tmp)
	if _, err := client.Execute("screendump", map[string]any{"filename": tmp}); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return nil, fmt.Errorf("screendump output missing: %w", err)
	}
	return PPMToPNG(data)
}

// Stop terminates the VM.
func (i *Instance) Stop() error {
	i.mu.Lock()
	client := i.client
	i.client = nil
	i.mu.Unlock()
	if client != nil {
		_ = client.Close()
	}
	if i.cmd.Process != nil {
		return i.cmd.Process.Kill()
	}
	return nil
}

// Running reports whether the VM process is alive.
func (i *Instance) Running() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.exitCode == nil && i.cmd.Process != nil
}

// Pid returns the process id.
func (i *Instance) Pid() int {
	if i.cmd.Process != nil {
		return i.cmd.Process.Pid
	}
	return 0
}

// LastError returns the most recent error string.
func (i *Instance) LastError() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.lastErr
}

// ExitCode returns the exit code once the process has ended.
func (i *Instance) ExitCode() *int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.exitCode
}

// StartedAt returns the launch time.
func (i *Instance) StartedAt() time.Time { return i.startedAt }

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func accelFlag() string {
	switch runtime.GOOS {
	case "linux":
		return "kvm"
	case "darwin":
		return "hvf"
	case "windows":
		return "whpx"
	default:
		return ""
	}
}
