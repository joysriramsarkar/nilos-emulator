// Package dap implements a Debug Adapter Protocol server for the emulator. It
// speaks standard DAP (Content-Length framed JSON over stdio), so it plugs
// into VS Code, Neovim, Emacs, etc. without a custom client.
//
// The debuggee is provided by a Backend. MockBackend is a self-contained
// program used for standalone use and tests; HTTPBackend attaches to a running
// nilemu control API. The real NilLang VM debugger is served by `nil dap`
// (see the nilLang repository), which nildap can proxy to — that is what makes
// source-line stepping real.
package dap

import (
	"io"
	"sync"

	"github.com/joysriramsarkar/nilos-emulator/internal/jsonwire"
)

// Thread, Frame, Scope, Variable and Breakpoint mirror DAP body types.
type Thread struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Frame struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type Scope struct {
	Name               string `json:"name"`
	VariablesReference int    `json:"variablesReference"`
	Expensive          bool   `json:"expensive"`
}

type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

type Breakpoint struct {
	ID       int    `json:"id"`
	Verified bool   `json:"verified"`
	Line     int    `json:"line"`
	Source   string `json:"source"`
}

// Backend is the debuggee.
type Backend interface {
	Threads() []Thread
	StackTrace(threadID int) ([]Frame, error)
	Scopes(frameID int) ([]Scope, error)
	Variables(ref int) ([]Variable, error)
	SetBreakpoints(source string, lines []int) ([]Breakpoint, error)
	Continue() error
	Next() error
	StepIn() error
	StepOut() error
	Pause() error
	Evaluate(expr string) (string, error)
	Close() error
}

// Server is a DAP adapter.
type Server struct {
	conn    *jsonwire.Conn
	backend Backend

	mu    sync.Mutex
	seq   int
	pause bool
}

// NewServer creates a DAP adapter over r/w.
func NewServer(r io.Reader, w io.Writer, backend Backend) *Server {
	return &Server{conn: jsonwire.NewConn(r, w), backend: backend}
}

func (s *Server) nextSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Server) respond(reqSeq int, command string, success bool, body any, message string) {
	msg := map[string]any{
		"seq": s.nextSeq(), "type": "response", "request_seq": reqSeq,
		"success": success, "command": command,
	}
	if body != nil {
		msg["body"] = body
	}
	if message != "" {
		msg["message"] = message
	}
	_ = s.conn.Write(msg)
}

func (s *Server) event(name string, body any) {
	msg := map[string]any{"seq": s.nextSeq(), "type": "event", "event": name}
	if body != nil {
		msg["body"] = body
	}
	_ = s.conn.Write(msg)
}

// Run processes DAP requests until disconnect or EOF.
func (s *Server) Run() error {
	for {
		msg, err := s.conn.Read()
		if err != nil {
			return err
		}
		command, _ := msg["command"].(string)
		reqSeq := intOf(msg["seq"])
		args, _ := msg["arguments"].(map[string]any)

		switch command {
		case "initialize":
			s.respond(reqSeq, command, true, map[string]any{
				"capabilities": map[string]any{
					"supportsConfigurationDoneRequest": true,
					"supportsEvaluateForHovers":        true,
					"supportsSetVariable":              false,
				},
				"serverInfo": map[string]any{"name": "nildap", "version": "1.0.0"},
			}, "")
			s.event("initialized", nil)
		case "launch", "attach":
			s.respond(reqSeq, command, true, nil, "")
		case "setBreakpoints":
			source := sourcePath(args)
			lines := breakpointLines(args)
			bps, err := s.backend.SetBreakpoints(source, lines)
			if err != nil {
				s.respond(reqSeq, command, false, nil, err.Error())
				continue
			}
			out := make([]map[string]any, 0, len(bps))
			for _, bp := range bps {
				out = append(out, map[string]any{
					"id": bp.ID, "verified": bp.Verified, "line": bp.Line,
					"source": map[string]any{"name": bp.Source, "path": bp.Source},
				})
			}
			s.respond(reqSeq, command, true, map[string]any{"breakpoints": out}, "")
		case "setExceptionBreakpoints":
			s.respond(reqSeq, command, true, map[string]any{"breakpoints": []any{}}, "")
		case "configurationDone":
			s.respond(reqSeq, command, true, nil, "")
			s.event("stopped", map[string]any{"reason": "entry", "threadId": 1, "allThreadsStopped": true})
		case "threads":
			threads := s.backend.Threads()
			s.respond(reqSeq, command, true, map[string]any{"threads": threads}, "")
		case "stackTrace":
			threadID := intOf(args["threadId"])
			frames, err := s.backend.StackTrace(threadID)
			if err != nil {
				s.respond(reqSeq, command, false, nil, err.Error())
				continue
			}
			out := make([]map[string]any, 0, len(frames))
			for _, f := range frames {
				out = append(out, map[string]any{
					"id": f.ID, "name": f.Name, "line": f.Line, "column": f.Column,
					"source": map[string]any{"name": f.Source, "path": f.Source},
				})
			}
			s.respond(reqSeq, command, true, map[string]any{"stackFrames": out, "totalFrames": len(out)}, "")
		case "scopes":
			scopes, err := s.backend.Scopes(intOf(args["frameId"]))
			if err != nil {
				s.respond(reqSeq, command, false, nil, err.Error())
				continue
			}
			s.respond(reqSeq, command, true, map[string]any{"scopes": scopes}, "")
		case "variables":
			vars, err := s.backend.Variables(intOf(args["variablesReference"]))
			if err != nil {
				s.respond(reqSeq, command, false, nil, err.Error())
				continue
			}
			s.respond(reqSeq, command, true, map[string]any{"variables": vars}, "")
		case "evaluate":
			expr, _ := args["expression"].(string)
			val, err := s.backend.Evaluate(expr)
			if err != nil {
				s.respond(reqSeq, command, false, nil, err.Error())
				continue
			}
			s.respond(reqSeq, command, true, map[string]any{"result": val, "variablesReference": 0}, "")
		case "continue":
			_ = s.backend.Continue()
			s.respond(reqSeq, command, true, map[string]any{"allThreadsContinued": true}, "")
			s.event("continued", map[string]any{"threadId": 1, "allThreadsContinued": true})
			s.event("stopped", map[string]any{"reason": "breakpoint", "threadId": 1, "allThreadsStopped": true})
		case "next", "stepIn", "stepOut":
			switch command {
			case "next":
				_ = s.backend.Next()
			case "stepIn":
				_ = s.backend.StepIn()
			case "stepOut":
				_ = s.backend.StepOut()
			}
			s.respond(reqSeq, command, true, nil, "")
			s.event("stopped", map[string]any{"reason": "step", "threadId": 1, "allThreadsStopped": true})
		case "pause":
			_ = s.backend.Pause()
			s.respond(reqSeq, command, true, nil, "")
			s.event("stopped", map[string]any{"reason": "pause", "threadId": 1, "allThreadsStopped": true})
		case "disconnect", "terminate":
			_ = s.backend.Close()
			s.respond(reqSeq, command, true, nil, "")
			s.event("terminated", nil)
			return nil
		default:
			s.respond(reqSeq, command, false, nil, "unsupported command: "+command)
		}
	}
}

func sourcePath(args map[string]any) string {
	if src, ok := args["source"].(map[string]any); ok {
		if p, ok := src["path"].(string); ok {
			return p
		}
		if n, ok := src["name"].(string); ok {
			return n
		}
	}
	return "main.nil"
}

func breakpointLines(args map[string]any) []int {
	var out []int
	if bps, ok := args["breakpoints"].([]any); ok {
		for _, b := range bps {
			if m, ok := b.(map[string]any); ok {
				out = append(out, intOf(m["line"]))
			}
		}
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
