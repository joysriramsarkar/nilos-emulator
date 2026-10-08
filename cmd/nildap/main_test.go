package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain doubles as a helper-process entry point: when NILDAP_ECHO_HELPER is
// set the test binary just copies stdin to stdout, so runProxy can be tested
// against a real child process without depending on any external command.
func TestMain(m *testing.M) {
	if os.Getenv("NILDAP_ECHO_HELPER") == "1" {
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"nil dap main.nil", []string{"nil", "dap", "main.nil"}},
		{"  nil   dap   ", []string{"nil", "dap"}},
		{`"C:\programming language\nil.exe" dap main.nil`,
			[]string{`C:\programming language\nil.exe`, "dap", "main.nil"}},
		{`nil dap "my program.nil"`, []string{"nil", "dap", "my program.nil"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitCommandLine(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("splitCommandLine(%q) = %#v, want %#v", tt.in, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitCommandLine(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDapCommandLine(t *testing.T) {
	got := dapCommandLine(`/programming language/nil.exe`, `/my prog/main.nil`)
	if got != `"/programming language/nil.exe" dap "/my prog/main.nil"` {
		t.Errorf("unexpected: %s", got)
	}
	// Paths without spaces must not be quoted.
	if got := dapCommandLine(`C:\nil.exe`, `main.nil`); got != `C:\nil.exe dap main.nil` {
		t.Errorf("unexpected: %s", got)
	}
}

// TestRunProxyPassthrough proves the child's stdout reaches ours and that stdin
// is forwarded, i.e. nildap is a transparent DAP bridge.
func TestRunProxyPassthrough(t *testing.T) {
	t.Setenv("NILDAP_ECHO_HELPER", "1")

	payload := "Content-Length: 11\r\n\r\n{\"ok\":true}"
	var out bytes.Buffer
	if err := runProxy(quote(os.Args[0]), strings.NewReader(payload), &out, io.Discard); err != nil {
		t.Fatalf("runProxy: %v", err)
	}
	if out.String() != payload {
		t.Errorf("stdout not passed through:\n got %q\nwant %q", out.String(), payload)
	}
}

// TestRunProxyPropagatesExitCode ensures a failing debuggee surfaces as a
// failure rather than being swallowed.
func TestRunProxyPropagatesExitCode(t *testing.T) {
	if err := runProxy("go nildap-nonexistent-subcommand", strings.NewReader(""), io.Discard, io.Discard); err == nil {
		t.Fatal("expected non-nil error for failing child")
	}
}

// TestRunProxyEmptyCommand guards the flag handler.
func TestRunProxyEmptyCommand(t *testing.T) {
	if err := runProxy("   ", strings.NewReader(""), io.Discard, io.Discard); err == nil {
		t.Fatal("expected an error for an empty command")
	}
}

// TestRealVMEndToEnd drives nildap's proxy mode against the real NilLang DAP
// server as an interactive client: it waits for each stop before sending the
// next request, exactly as an editor would.
//
// Requires NIL_BIN to point at a built `nil` binary; skipped otherwise.
func TestRealVMEndToEnd(t *testing.T) {
	nilBin := os.Getenv("NIL_BIN")
	if nilBin == "" {
		t.Skip("set NIL_BIN to a built nil binary to run this test")
	}
	if _, err := os.Stat(nilBin); err != nil {
		t.Fatalf("NIL_BIN: %v", err)
	}

	srcPath := filepath.Join(t.TempDir(), "main.nil")
	if err := os.WriteFile(srcPath, []byte("let a = 1;\nlet b = 2;\nlet c = a + b;\nc;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdinR, stdinW := io.Pipe()
	out := &safeBuf{}
	var (
		childErr error
		errBuf   bytes.Buffer
	)
	exited := make(chan struct{})
	go func() {
		childErr = runProxy(dapCommandLine(nilBin, srcPath), stdinR, out, &errBuf)
		close(exited)
	}()
	// Always unblock the child if the assertions below fail early.
	defer func() {
		_ = stdinW.Close()
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
			t.Error("child still running after test finished")
		}
	}()

	send := func(seq int, command string, args any) {
		t.Helper()
		if _, err := io.WriteString(stdinW, dapFrame(seq, command, args)); err != nil {
			t.Fatalf("write %s: %v", command, err)
		}
	}
	wait := func(what string, pred func([]map[string]any) bool) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if pred(parseDAPStream(t, out.Bytes())) {
				return
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatalf("timeout waiting for %s\n--- session ---\n%s", what, out.String())
	}

	send(1, "initialize", map[string]any{})
	wait("initialize capabilities", func(ms []map[string]any) bool {
		for _, m := range ms {
			if m["type"] == "response" && m["command"] == "initialize" {
				body, _ := m["body"].(map[string]any)
				_, ok := body["capabilities"]
				return ok
			}
		}
		return false
	})

	send(2, "launch", map[string]any{"program": srcPath})
	send(3, "setBreakpoints", map[string]any{
		"source":      map[string]any{"path": srcPath},
		"breakpoints": []any{map[string]any{"line": 3}},
	})
	send(4, "configurationDone", nil)
	wait("entry stop", stoppedWith("entry", 0))

	send(5, "continue", map[string]any{"threadId": 1})
	wait("breakpoint on line 3", stoppedWith("breakpoint", 3))

	send(6, "variables", map[string]any{"variablesReference": 2})
	wait("globals a=1 b=2", func(ms []map[string]any) bool {
		for _, m := range ms {
			if m["type"] == "response" && m["command"] == "variables" && m["success"] == true {
				body, _ := m["body"].(map[string]any)
				vars, _ := body["variables"].([]any)
				got := map[string]string{}
				for _, v := range vars {
					vm, _ := v.(map[string]any)
					got[vm["name"].(string)] = vm["value"].(string)
				}
				return got["a"] == "1" && got["b"] == "2"
			}
		}
		return false
	})

	send(7, "disconnect", map[string]any{})
	_ = stdinW.Close()
	select {
	case <-exited:
		if childErr != nil {
			t.Errorf("proxy exited with error: %v\n%s", childErr, errBuf.String())
		}
	case <-time.After(20 * time.Second):
		t.Error("child did not exit after disconnect")
	}
}

// stoppedWith matches a stopped event with the given reason (and line, when
// line > 0).
func stoppedWith(reason string, line int) func([]map[string]any) bool {
	return func(ms []map[string]any) bool {
		for _, m := range ms {
			if m["type"] == "event" && m["event"] == "stopped" {
				b, _ := m["body"].(map[string]any)
				if b["reason"] != reason {
					continue
				}
				if line > 0 && intOf(b["line"]) != line {
					continue
				}
				return true
			}
		}
		return false
	}
}

// safeBuf is a buffer safe to read while another goroutine writes to it.
type safeBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuf) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.b.Bytes()...)
}

func (s *safeBuf) String() string { return string(s.Bytes()) }

// ─── helpers ─────────────────────────────────────────────────────────────

func quote(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

func dapFrame(seq int, command string, args any) string {
	m := map[string]any{"seq": seq, "type": "request", "command": command}
	if args != nil {
		m["arguments"] = args
	}
	data, _ := json.Marshal(m)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
}

func parseDAPStream(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	rest := data
	for len(rest) > 0 {
		idx := bytes.Index(rest, []byte("\r\n\r\n"))
		if idx < 0 {
			break
		}
		header := string(rest[:idx])
		length := -1
		for _, line := range strings.Split(header, "\r\n") {
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			}
		}
		if length < 0 || idx+4+length > len(rest) {
			break
		}
		body := rest[idx+4 : idx+4+length]
		rest = rest[idx+4+length:]
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("bad frame: %v: %s", err, body)
		}
		out = append(out, m)
	}
	return out
}

func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
