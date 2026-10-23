package dap

import (
	"fmt"
	"strings"
)

// MockBackend is a small deterministic NilLang-like program used for
// standalone debugging and tests. It implements the same Backend contract the
// real runtime adapter will.
type MockBackend struct {
	source      string
	line        int
	count       int
	breakpoints map[int]bool
	nextBpID    int
}

// NewMockBackend creates the sample program.
func NewMockBackend() *MockBackend {
	return &MockBackend{
		source:      "main.nil",
		line:        3,
		count:       0,
		breakpoints: map[int]bool{},
		nextBpID:    1,
	}
}

// Lines of the mock program.
var mockLines = []string{
	"@Component",
	"ফাংশন Counter() {",
	"  চল count = 0;",
	"  count = count + 1;",
	"  প্রিন্ট(count);",
	"}",
}

func (m *MockBackend) Threads() []Thread { return []Thread{{ID: 1, Name: "main"}} }

func (m *MockBackend) StackTrace(threadID int) ([]Frame, error) {
	return []Frame{
		{ID: 1, Name: "Counter()", Source: m.source, Line: m.line, Column: 1},
		{ID: 2, Name: "main()", Source: m.source, Line: 1, Column: 1},
	}, nil
}

func (m *MockBackend) Scopes(frameID int) ([]Scope, error) {
	return []Scope{
		{Name: "Locals", VariablesReference: 1},
		{Name: "Emulator", VariablesReference: 2},
	}, nil
}

func (m *MockBackend) Variables(ref int) ([]Variable, error) {
	switch ref {
	case 1:
		return []Variable{
			{Name: "count", Value: fmt.Sprintf("%d", m.count), Type: "int"},
			{Name: "currentLine", Value: fmt.Sprintf("%d", m.line), Type: "int"},
		}, nil
	case 2:
		return []Variable{
			{Name: "runtime", Value: "nilemu-mockvm/1.0", Type: "string"},
			{Name: "protocol", Value: "NilHost v1", Type: "string"},
		}, nil
	}
	return nil, fmt.Errorf("unknown variablesReference %d", ref)
}

func (m *MockBackend) SetBreakpoints(source string, lines []int) ([]Breakpoint, error) {
	if source != "" {
		m.source = source
	}
	m.breakpoints = map[int]bool{}
	out := make([]Breakpoint, 0, len(lines))
	for _, ln := range lines {
		m.breakpoints[ln] = true
		out = append(out, Breakpoint{ID: m.nextBpID, Verified: ln >= 1 && ln <= len(mockLines), Line: ln, Source: m.source})
		m.nextBpID++
	}
	return out, nil
}

func (m *MockBackend) Continue() error {
	// Run until the next breakpoint or program end.
	for i := 0; i < len(mockLines)*2; i++ {
		m.step()
		if m.breakpoints[m.line] {
			return nil
		}
	}
	return nil
}

func (m *MockBackend) Next() error   { m.step(); return nil }
func (m *MockBackend) StepIn() error { m.step(); return nil }
func (m *MockBackend) StepOut() error {
	m.line++
	if m.line > len(mockLines) {
		m.line = 3
	}
	return nil
}

func (m *MockBackend) step() {
	m.line++
	if m.line > len(mockLines)-1 {
		m.line = 3
	}
	if m.line == 4 {
		m.count++
	}
}

func (m *MockBackend) Pause() error { return nil }

func (m *MockBackend) Evaluate(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	switch expr {
	case "count":
		return fmt.Sprintf("%d", m.count), nil
	case "line", "currentLine":
		return fmt.Sprintf("%d", m.line), nil
	case "":
		return "", fmt.Errorf("empty expression")
	}
	if strings.Contains(expr, "count") {
		return fmt.Sprintf("%d", m.count), nil
	}
	return "(nil)", nil
}

func (m *MockBackend) Close() error { return nil }
