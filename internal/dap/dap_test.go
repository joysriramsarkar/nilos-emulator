package dap

import (
	"net"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/jsonwire"
)

type client struct {
	conn   *jsonwire.Conn
	events []map[string]any
	raw    net.Conn
}

// dialPair connects a server over an OS-buffered TCP loopback so neither side
// deadlocks when one writes while the other is momentarily not reading.
func dialPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c
		}
	}()
	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	srv := <-accepted
	ln.Close()
	return srv, cli
}

func newClient(t *testing.T, backend Backend) *client {
	t.Helper()
	srv, cli := dialPair(t)
	s := NewServer(srv, srv, backend)
	go func() { _ = s.Run() }()
	t.Cleanup(func() { srv.Close(); cli.Close() })
	return &client{conn: jsonwire.NewConn(cli, cli), raw: cli}
}

// request sends a command and returns the matching response, buffering events.
func (c *client) request(t *testing.T, seq int, command string, args map[string]any) map[string]any {
	t.Helper()
	msg := map[string]any{"seq": seq, "type": "request", "command": command}
	if args != nil {
		msg["arguments"] = args
	}
	if err := c.conn.Write(msg); err != nil {
		t.Fatalf("write %s: %v", command, err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m, err := c.conn.Read()
		if err != nil {
			t.Fatalf("read for %s: %v", command, err)
		}
		if m["type"] == "event" {
			c.events = append(c.events, m)
			continue
		}
		if m["type"] == "response" && int(m["request_seq"].(float64)) == seq {
			return m
		}
	}
	t.Fatalf("no response for %s", command)
	return nil
}

func body(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	b, _ := resp["body"].(map[string]any)
	if b == nil {
		t.Fatalf("response has no body: %+v", resp)
	}
	return b
}

func TestDAPFlow(t *testing.T) {
	c := newClient(t, NewMockBackend())

	resp := c.request(t, 1, "initialize", map[string]any{})
	if resp["success"] != true {
		t.Fatalf("initialize failed: %+v", resp)
	}
	if _, ok := body(t, resp)["capabilities"].(map[string]any); !ok {
		t.Fatalf("initialize response missing capabilities: %+v", resp)
	}

	resp = c.request(t, 2, "setBreakpoints", map[string]any{
		"source":      map[string]any{"path": "main.nil"},
		"breakpoints": []any{map[string]any{"line": 4}},
	})
	bps, _ := body(t, resp)["breakpoints"].([]any)
	if len(bps) != 1 {
		t.Fatalf("expected 1 breakpoint, got %+v", bps)
	}

	c.request(t, 3, "configurationDone", nil)

	resp = c.request(t, 4, "threads", nil)
	threads, _ := body(t, resp)["threads"].([]any)
	if len(threads) == 0 {
		t.Fatal("no threads")
	}

	resp = c.request(t, 5, "stackTrace", map[string]any{"threadId": 1})
	frames, _ := body(t, resp)["stackFrames"].([]any)
	if len(frames) == 0 {
		t.Fatal("no stack frames")
	}

	resp = c.request(t, 6, "scopes", map[string]any{"frameId": 1})
	scopes, _ := body(t, resp)["scopes"].([]any)
	if len(scopes) == 0 {
		t.Fatal("no scopes")
	}

	resp = c.request(t, 7, "variables", map[string]any{"variablesReference": 1})
	vars, _ := body(t, resp)["variables"].([]any)
	foundCount := false
	for _, v := range vars {
		if m, ok := v.(map[string]any); ok && m["name"] == "count" {
			foundCount = true
		}
	}
	if !foundCount {
		t.Fatalf("count variable not found: %+v", vars)
	}

	resp = c.request(t, 8, "next", nil)
	if resp["success"] != true {
		t.Fatal("next failed")
	}

	resp = c.request(t, 9, "evaluate", map[string]any{"expression": "count"})
	if result, _ := body(t, resp)["result"].(string); result == "" {
		t.Fatal("evaluate returned empty")
	}

	resp = c.request(t, 10, "disconnect", nil)
	if resp["success"] != true {
		t.Fatal("disconnect failed")
	}

	// We should have observed the stopped/continued/terminated events.
	sawStopped := false
	for _, e := range c.events {
		if e["event"] == "stopped" {
			sawStopped = true
		}
	}
	if !sawStopped {
		t.Fatal("expected a stopped event")
	}
}

func TestMockBackendStepAndEvaluate(t *testing.T) {
	m := NewMockBackend()
	_ = m.Next()
	if v, _ := m.Evaluate("count"); v == "" {
		t.Fatal("empty evaluate")
	}
	if _, err := m.Variables(999); err == nil {
		t.Fatal("expected error for unknown reference")
	}
}
