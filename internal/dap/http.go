package dap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// HTTPBackend attaches the DAP adapter to a running nilemu control API.
type HTTPBackend struct {
	base   string
	device string
	app    string
	token  string
	client *http.Client
}

// NewHTTPBackend creates a backend for a device/app on a control API.
func NewHTTPBackend(base, device, app string) *HTTPBackend {
	return &HTTPBackend{
		base:   base,
		device: device,
		app:    app,
		token:  os.Getenv("NILEMU_TOKEN"),
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (b *HTTPBackend) url(path string, q url.Values) string {
	u := fmt.Sprintf("%s/api/v1/devices/%s/apps/%s/debug/%s", b.base, b.device, b.app, path)
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

func (b *HTTPBackend) get(path string, q url.Values, out any) error {
	req, err := http.NewRequest(http.MethodGet, b.url(path, q), nil)
	if err != nil {
		return err
	}
	return b.do(req, out)
}

func (b *HTTPBackend) post(path string, body any, out any) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, b.url(path, nil), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return b.do(req, out)
}

func (b *HTTPBackend) do(req *http.Request, out any) error {
	if b.token != "" {
		req.Header.Set("X-Nilemu-Token", b.token)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, string(raw))
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (b *HTTPBackend) Threads() []Thread { return []Thread{{ID: 1, Name: "main"}} }

func (b *HTTPBackend) StackTrace(threadID int) ([]Frame, error) {
	var res struct {
		StackFrames []Frame `json:"stackFrames"`
	}
	if err := b.get("stack", nil, &res); err != nil {
		return nil, err
	}
	return res.StackFrames, nil
}

func (b *HTTPBackend) Scopes(frameID int) ([]Scope, error) {
	var res struct {
		Scopes []Scope `json:"scopes"`
	}
	if err := b.get("scopes", nil, &res); err != nil {
		return nil, err
	}
	return res.Scopes, nil
}

func (b *HTTPBackend) Variables(ref int) ([]Variable, error) {
	var res struct {
		Variables []Variable `json:"variables"`
	}
	q := url.Values{"ref": {fmt.Sprintf("%d", ref)}}
	if err := b.get("variables", q, &res); err != nil {
		return nil, err
	}
	return res.Variables, nil
}

func (b *HTTPBackend) SetBreakpoints(source string, lines []int) ([]Breakpoint, error) {
	var res struct {
		Breakpoints []Breakpoint `json:"breakpoints"`
	}
	err := b.post("breakpoints", map[string]any{"source": source, "lines": lines}, &res)
	return res.Breakpoints, err
}

func (b *HTTPBackend) command(name string) error {
	return b.post("command", map[string]any{"command": name}, nil)
}

func (b *HTTPBackend) Continue() error { return b.command("continue") }
func (b *HTTPBackend) Next() error     { return b.command("next") }
func (b *HTTPBackend) StepIn() error   { return b.command("stepIn") }
func (b *HTTPBackend) StepOut() error  { return b.command("stepOut") }
func (b *HTTPBackend) Pause() error    { return b.command("pause") }

func (b *HTTPBackend) Evaluate(expr string) (string, error) {
	var res struct {
		Result string `json:"result"`
	}
	q := url.Values{"expr": {expr}}
	if err := b.get("evaluate", q, &res); err != nil {
		return "", err
	}
	return res.Result, nil
}

func (b *HTTPBackend) Close() error { return nil }
