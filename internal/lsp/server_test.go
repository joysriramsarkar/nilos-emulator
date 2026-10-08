package lsp

import (
	"net"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/jsonwire"
)

type lspClient struct {
	conn *jsonwire.Conn
}

func newLSPClient(t *testing.T) *lspClient {
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
	s := NewServer(srv, srv)
	go func() { _ = s.Run() }()
	t.Cleanup(func() { srv.Close(); cli.Close() })
	return &lspClient{conn: jsonwire.NewConn(cli, cli)}
}

func (c *lspClient) send(t *testing.T, msg map[string]any) {
	t.Helper()
	if err := c.conn.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (c *lspClient) read(t *testing.T) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m, err := c.conn.Read()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return m
	}
	t.Fatal("read timeout")
	return nil
}

func TestLSPInitializeDiagnosticsCompletion(t *testing.T) {
	c := newLSPClient(t)

	c.send(t, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	resp := c.read(t)
	if resp["id"] == nil {
		t.Fatalf("no initialize response: %+v", resp)
	}
	result, _ := resp["result"].(map[string]any)
	caps, _ := result["capabilities"].(map[string]any)
	if caps == nil {
		t.Fatalf("no capabilities: %+v", resp)
	}

	// Opening broken code must publish diagnostics.
	c.send(t, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri": "file:///main.nil", "text": "ফাংশন Test() {\n  চল x\n",
		}},
	})
	diagMsg := c.read(t)
	if diagMsg["method"] != "textDocument/publishDiagnostics" {
		t.Fatalf("expected diagnostics notification, got %+v", diagMsg)
	}
	params, _ := diagMsg["params"].(map[string]any)
	diags, _ := params["diagnostics"].([]any)
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}

	// Completion request.
	c.send(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/completion",
		"params": map[string]any{"textDocument": map[string]any{"uri": "file:///main.nil"}, "position": map[string]any{"line": 0, "character": 0}}})
	for {
		m := c.read(t)
		if m["id"] != nil {
			items, _ := m["result"].([]any)
			if len(items) == 0 {
				t.Fatalf("expected completions, got %+v", m)
			}
			break
		}
	}

	// Hover request.
	c.send(t, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/hover",
		"params": map[string]any{"textDocument": map[string]any{"uri": "file:///main.nil"}, "position": map[string]any{"line": 0, "character": 2}}})
	for {
		m := c.read(t)
		if m["id"] != nil {
			if m["result"] == nil {
				t.Fatal("expected hover result")
			}
			break
		}
	}

	c.send(t, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "shutdown"})
	for {
		m := c.read(t)
		if m["id"] != nil {
			break
		}
	}
	c.send(t, map[string]any{"jsonrpc": "2.0", "method": "exit"})
}

func TestAnalyzeDetectsUnbalanced(t *testing.T) {
	diags := Analyze("ফাংশন Test() {\n  চল x\n")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for unbalanced code")
	}
	if diags := Analyze("চল x = 1;\n"); len(diags) != 0 {
		t.Fatalf("expected no diagnostics for valid code, got %+v", diags)
	}
}
