package lsp

import (
	"io"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/joysriramsarkar/nilos-emulator/internal/jsonwire"
)

// Server is a minimal but functional NilLang language server.
type Server struct {
	conn *jsonwire.Conn
	mu   sync.Mutex
	docs map[string]string
}

// NewServer creates a server reading/writing framed JSON.
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{conn: jsonwire.NewConn(r, w), docs: map[string]string{}}
}

// Run processes messages until exit or EOF.
func (s *Server) Run() error {
	for {
		raw, err := s.conn.Read()
		if err != nil {
			return err
		}
		method, _ := raw["method"].(string)
		id, hasID := raw["id"]
		params, _ := raw["params"].(map[string]any)

		switch method {
		case "initialize":
			s.reply(id, map[string]any{
				"capabilities": map[string]any{
					"textDocumentSync":       1, // full
					"completionProvider":     map[string]any{"triggerCharacters": []string{"@", "."}},
					"hoverProvider":          true,
					"documentSymbolProvider": true,
				},
				"serverInfo": map[string]any{"name": "nilsp", "version": "1.0.0"},
			})
		case "initialized":
			// notification
		case "shutdown":
			s.reply(id, nil)
		case "exit":
			return nil
		case "textDocument/didOpen":
			uri, text := docAndText(params)
			s.mu.Lock()
			s.docs[uri] = text
			s.mu.Unlock()
			s.publishDiagnostics(uri, text)
		case "textDocument/didChange":
			uri, text := docAndText(params)
			s.mu.Lock()
			s.docs[uri] = text
			s.mu.Unlock()
			s.publishDiagnostics(uri, text)
		case "textDocument/didClose":
			uri, _ := docAndText(params)
			s.mu.Lock()
			delete(s.docs, uri)
			s.mu.Unlock()
		case "textDocument/completion":
			items := Completions()
			out := make([]map[string]any, 0, len(items))
			for _, it := range items {
				out = append(out, map[string]any{
					"label": it.Label, "kind": lspCompletionKind(it.Kind),
					"detail": it.Detail, "documentation": it.Documentation,
					"insertText": it.InsertText, "insertTextFormat": 2,
				})
			}
			s.reply(id, out)
		case "textDocument/hover":
			uri, _ := docAndText(params)
			s.mu.Lock()
			text := s.docs[uri]
			s.mu.Unlock()
			word := wordAt(text, position(params))
			info := Hover(word)
			s.reply(id, map[string]any{
				"contents": map[string]any{"kind": "markdown", "value": "**" + info.Signature + "**\n\n" + info.Description},
			})
		case "textDocument/documentSymbol":
			uri, _ := docAndText(params)
			s.mu.Lock()
			text := s.docs[uri]
			s.mu.Unlock()
			s.reply(id, symbols(text))
		default:
			if hasID {
				s.reply(id, nil)
			}
		}
	}
}

func (s *Server) reply(id any, result any) {
	if id == nil {
		return
	}
	_ = s.conn.Write(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (s *Server) publishDiagnostics(uri, text string) {
	diags := Analyze(text)
	out := make([]map[string]any, 0, len(diags))
	for _, d := range diags {
		out = append(out, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": max(0, d.Line-1), "character": max(0, d.Column-1)},
				"end":   map[string]any{"line": max(0, d.Line-1), "character": max(0, d.Column)},
			},
			"severity": lspSeverity(d.Severity),
			"source":   d.Source,
			"message":  d.Message,
		})
	}
	_ = s.conn.Write(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics",
		"params": map[string]any{"uri": uri, "diagnostics": out},
	})
}

func docAndText(params map[string]any) (string, string) {
	if params == nil {
		return "", ""
	}
	td, _ := params["textDocument"].(map[string]any)
	uri, _ := td["uri"].(string)
	text, _ := td["text"].(string)
	if tcc, ok := params["contentChanges"].([]any); ok && len(tcc) > 0 {
		if first, ok := tcc[0].(map[string]any); ok {
			if t, ok := first["text"].(string); ok {
				text = t
			}
		}
	}
	return uri, text
}

func position(params map[string]any) map[string]any {
	if p, ok := params["position"].(map[string]any); ok {
		return p
	}
	return nil
}

func wordAt(text string, pos map[string]any) string {
	if pos == nil {
		return ""
	}
	lineF, _ := pos["line"].(float64)
	charF, _ := pos["character"].(float64)
	lines := strings.Split(text, "\n")
	li := int(lineF)
	if li < 0 || li >= len(lines) {
		return ""
	}
	line := lines[li]
	ci := int(charF)
	if ci > len(line) {
		ci = len(line)
	}
	runes := []rune(line)
	ri := len([]rune(line[:ci]))
	start := ri
	for start > 0 && isIdent(runes[start-1]) {
		start--
	}
	end := ri
	for end < len(runes) && isIdent(runes[end]) {
		end++
	}
	return string(runes[start:end])
}

func isIdent(r rune) bool {
	return r == '_' || r == '@' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

var funcRe = regexp.MustCompile(`(?:ফাংশন|function)\s+([^\s(]+)`)

func symbols(text string) []map[string]any {
	var out []map[string]any
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@Component") {
			out = append(out, map[string]any{
				"name": "@Component", "kind": 5, // Class
				"location": map[string]any{"uri": "", "range": rng(i, 0, i, len(line))},
			})
		}
		if m := funcRe.FindStringSubmatch(trimmed); m != nil {
			out = append(out, map[string]any{
				"name": m[1], "kind": 12, // Function
				"location": map[string]any{"uri": "", "range": rng(i, 0, i, len(line))},
			})
		}
	}
	return out
}

func rng(l0, c0, l1, c1 int) map[string]any {
	return map[string]any{
		"start": map[string]any{"line": l0, "character": c0},
		"end":   map[string]any{"line": l1, "character": c1},
	}
}

func lspSeverity(s string) int {
	switch s {
	case "error":
		return 1
	case "warning":
		return 2
	case "info":
		return 3
	default:
		return 4
	}
}

func lspCompletionKind(k string) int {
	switch k {
	case "keyword":
		return 14
	case "function":
		return 3
	case "variable":
		return 6
	case "component":
		return 7
	default:
		return 1
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
