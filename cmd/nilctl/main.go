// Command nilctl is the emulator control CLI — the adb analogue. It talks to
// a running nilemu control API over HTTP.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/render"
)

var (
	server = envOr("NILEMU_SERVER", "http://127.0.0.1:8085")
	token  = os.Getenv("NILEMU_TOKEN")
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	cmd := args[0]
	rest := args[1:]
	var err error
	switch cmd {
	case "devices", "list-devices":
		err = cmdDevices(rest)
	case "create":
		err = cmdCreate(rest)
	case "start":
		err = cmdDeviceAction(rest, "start")
	case "stop":
		err = cmdDeviceAction(rest, "stop")
	case "delete":
		err = cmdDelete(rest)
	case "wipe":
		err = cmdWipe(rest)
	case "install":
		err = cmdInstall(rest)
	case "apps":
		err = cmdApps(rest)
	case "launch":
		err = cmdAppAction(rest, "start")
	case "stop-app":
		err = cmdAppAction(rest, "stop")
	case "pause":
		err = cmdAppAction(rest, "pause")
	case "resume":
		err = cmdAppAction(rest, "resume")
	case "reload":
		err = cmdAppAction(rest, "reload")
	case "uninstall":
		err = cmdAppAction(rest, "uninstall")
	case "clear-data":
		err = cmdAppAction(rest, "clear-data")
	case "tree":
		err = cmdTree(rest)
	case "screenshot":
		err = cmdScreenshot(rest)
	case "input":
		err = cmdInput(rest)
	case "ime":
		err = cmdIME(rest)
	case "battery":
		err = cmdBattery(rest)
	case "net":
		err = cmdNet(rest)
	case "hal":
		err = cmdHAL(rest)
	case "fault":
		err = cmdFault(rest)
	case "record":
		err = cmdRecord(rest)
	case "recordings":
		err = cmdRecordings(rest)
	case "replay":
		err = cmdReplay(rest)
	case "logs":
		err = cmdLogs(rest)
	case "sessions":
		err = cmdGetJSON("/api/v1/sessions")
	case "bus":
		err = cmdGetJSON("/api/v1/bus/log?n=100")
	case "snapshot":
		err = cmdSnapshot(rest)
	case "snapshots":
		err = cmdSnapshots(rest)
	case "restore":
		err = cmdRestore(rest)
	case "health":
		err = cmdGetJSON("/api/v1/health")
	case "profiles":
		err = cmdGetJSON("/api/v1/profiles")
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`nilctl — NilOS device emulator control

  nilctl devices                          list devices
  nilctl profiles                         list device profiles
  nilctl create --profile <id> [--name N] create a device
  nilctl start <device> | stop <device> | delete <device> | wipe <device>
  nilctl install <bundle.nilax> --device <id>
  nilctl apps --device <id>
  nilctl launch <app> --device <id>       start an app
  nilctl stop-app|pause|resume|reload|uninstall|clear-data <app> --device <id>
  nilctl tree <app> --device <id>         print the current UI tree
  nilctl screenshot <out.svg> <app> --device <id>
  nilctl input tap <x> <y> --device <id> [--app <id>]
  nilctl input text "<text>" --device <id> [--app <id>]
  nilctl ime commit "<text>" --device <id> --app <id>
  nilctl battery <level> [--charging] --device <id>
  nilctl net <5g|4g|3g|edge|offline> --device <id>
  nilctl fault <dev> <op> [--code X] [--times N] --device <id>
  nilctl record start --device <id> --app <id> [--name n]
  nilctl record stop --device <id>
  nilctl recordings --device <id>
  nilctl replay <name> --device <id> [--app <id>] [--instant] [--speed N]
  nilctl logs [-n N]
  nilctl sessions | bus | health

Global: --server <url> (or $NILEMU_SERVER), --token <t> (or $NILEMU_TOKEN)
`)
}

// ─── command implementations ────────────────────────────────────────────

func cmdDevices(args []string) error {
	return cmdGetJSON("/api/v1/devices")
}

func cmdCreate(args []string) error {
	fs := newFlagSet("create")
	device := fs.String("device", "", "ignored")
	profile := fs.String("profile", "nilphone-ultra", "device profile id")
	name := fs.String("name", "", "device name")
	_ = device
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	body := map[string]any{"profile_id": *profile, "name": *name}
	return postJSON("/api/v1/devices", body, true)
}

func cmdDeviceAction(args []string, action string) error {
	fs := newFlagSet("")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	id := *device
	if id == "" && len(rest) > 0 {
		id = rest[0]
	}
	if id == "" {
		return fmt.Errorf("device id required")
	}
	return postJSON("/api/v1/devices/"+id+"/"+action, nil, false)
}

func cmdDelete(args []string) error {
	fs := newFlagSet("")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	id := *device
	if id == "" && len(fs.Args()) > 0 {
		id = fs.Args()[0]
	}
	return do("DELETE", "/api/v1/devices/"+id, nil, true)
}

func cmdWipe(args []string) error {
	return cmdDeviceAction(args, "wipe")
}

func cmdInstall(args []string) error {
	fs := newFlagSet("install")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("bundle path required")
	}
	if *device == "" {
		return fmt.Errorf("--device required")
	}
	return postJSON("/api/v1/devices/"+*device+"/install", map[string]any{"path": rest[0]}, true)
}

func cmdApps(args []string) error {
	fs := newFlagSet("apps")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return getJSON("/api/v1/devices/" + *device + "/apps")
}

func cmdAppAction(args []string, action string) error {
	fs := newFlagSet("")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if *device == "" || len(rest) == 0 {
		return fmt.Errorf("usage: <app> --device <id>")
	}
	return postJSON("/api/v1/devices/"+*device+"/apps/"+rest[0]+"/"+action, nil, false)
}

func cmdTree(args []string) error {
	fs := newFlagSet("tree")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	appID := *app
	if appID == "" && len(rest) > 0 {
		appID = rest[0]
	}
	return getJSON("/api/v1/devices/" + *device + "/apps/" + appID + "/tree")
}

func cmdScreenshot(args []string) error {
	fs := newFlagSet("screenshot")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("output path required")
	}
	out := rest[0]
	appID := *app
	if appID == "" && len(rest) > 1 {
		appID = rest[1]
	}
	raw, err := fetch("/api/v1/devices/" + *device + "/apps/" + appID + "/tree")
	if err != nil {
		return err
	}
	node, err := render.Parse(raw)
	if err != nil {
		return err
	}
	svg := render.SVG(node, render.Options{Width: 390, Height: 844, Theme: "dark"})
	return os.WriteFile(out, []byte(svg), 0o644)
}

func cmdInput(args []string) error {
	fs := newFlagSet("input")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("input kind required (tap|text|longpress|back)")
	}
	body := map[string]any{"kind": rest[0]}
	switch rest[0] {
	case "tap", "longpress":
		if len(rest) >= 3 {
			body["x"] = atof(rest[1])
			body["y"] = atof(rest[2])
		}
		body["phase"] = "up"
	case "text":
		if len(rest) >= 2 {
			body["text"] = strings.Join(rest[1:], " ")
		}
	case "back":
		body["phase"] = "up"
	}
	path := "/api/v1/devices/" + *device + "/input"
	if *app != "" {
		path = "/api/v1/devices/" + *device + "/apps/" + *app + "/input"
	}
	return postJSON(path, body, false)
}

func cmdIME(args []string) error {
	fs := newFlagSet("ime")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: ime <op> <text> --device <id> --app <id>")
	}
	body := map[string]any{"op": rest[0], "text": strings.Join(rest[1:], " ")}
	return postJSON("/api/v1/devices/"+*device+"/apps/"+*app+"/ime", body, false)
}

func cmdBattery(args []string) error {
	fs := newFlagSet("battery")
	device := fs.String("device", "", "device id")
	charging := fs.Bool("charging", false, "charging state")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("battery level required")
	}
	level := 0
	fmt.Sscanf(rest[0], "%d", &level)
	return postJSON("/api/v1/devices/"+*device+"/battery", map[string]any{"level": level, "charging": *charging}, true)
}

func cmdNet(args []string) error {
	fs := newFlagSet("net")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("network mode required")
	}
	return postJSON("/api/v1/devices/"+*device+"/network", map[string]any{"mode": rest[0]}, true)
}

func cmdHAL(args []string) error {
	fs := newFlagSet("hal")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return getJSON("/api/v1/devices/" + *device + "/hal")
}

func cmdFault(args []string) error {
	fs := newFlagSet("fault")
	device := fs.String("device", "", "device id")
	code := fs.String("code", "E_IO", "fault code")
	times := fs.Int("times", 1, "how many times")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: fault <dev> <op>")
	}
	body := map[string]any{"dev": rest[0], "op": rest[1], "code": *code, "times": *times}
	return postJSON("/api/v1/devices/"+*device+"/hal/fault", body, false)
}

func cmdLogs(args []string) error {
	fs := newFlagSet("logs")
	n := fs.Int("n", 50, "number of entries")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return getJSON(fmt.Sprintf("/api/v1/logs?n=%d", *n))
}

func cmdRecord(args []string) error {
	fs := newFlagSet("record")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	name := fs.String("name", "", "recording name")
	rest := reorderFlags(args)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	positional := fs.Args()
	action := "status"
	if len(positional) > 0 {
		action = positional[0]
	}
	switch action {
	case "start":
		if *device == "" {
			return fmt.Errorf("--device required")
		}
		return postJSON("/api/v1/devices/"+*device+"/record/start", map[string]any{"app": *app, "name": *name}, true)
	case "stop":
		return postJSON("/api/v1/devices/"+*device+"/record/stop", nil, true)
	case "status":
		return getJSON("/api/v1/devices/" + *device + "/record/status")
	default:
		return fmt.Errorf("unknown record action %q (start|stop|status)", action)
	}
}

func cmdRecordings(args []string) error {
	fs := newFlagSet("recordings")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return getJSON("/api/v1/devices/" + *device + "/recordings")
}

func cmdReplay(args []string) error {
	fs := newFlagSet("replay")
	device := fs.String("device", "", "device id")
	app := fs.String("app", "", "app id")
	instant := fs.Bool("instant", false, "ignore timing")
	speed := fs.Float64("speed", 1, "playback speed multiplier")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("recording name required")
	}
	body := map[string]any{"app": *app, "instant": *instant, "speed": *speed}
	return postJSON("/api/v1/devices/"+*device+"/recordings/"+rest[0]+"/replay", body, true)
}

func cmdSnapshot(args []string) error {
	fs := newFlagSet("snapshot")
	device := fs.String("device", "", "device id")
	name := fs.String("name", "snap", "snapshot name")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return postJSON("/api/v1/devices/"+*device+"/snapshots", map[string]any{"name": *name}, true)
}

func cmdSnapshots(args []string) error {
	fs := newFlagSet("snapshots")
	device := fs.String("device", "", "device id")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	return getJSON("/api/v1/devices/" + *device + "/snapshots")
}

func cmdRestore(args []string) error {
	fs := newFlagSet("restore")
	device := fs.String("device", "", "device id")
	name := fs.String("name", "", "snapshot name")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	n := *name
	if n == "" && len(rest) > 0 {
		n = rest[0]
	}
	return postJSON("/api/v1/devices/"+*device+"/snapshots/"+n+"/restore", nil, true)
}

func cmdGetJSON(path string) error { return getJSON(path) }

// ─── http helpers ───────────────────────────────────────────────────────

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&server, "server", server, "control API base URL")
	fs.StringVar(&token, "token", token, "control API token")
	fs.SetOutput(os.Stderr)
	return fs
}

// reorderFlags moves flag tokens (and their values) ahead of positional
// arguments, because Go's flag package stops parsing at the first positional.
func reorderFlags(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := a
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			continue
		}
		if name == "-charging" || name == "--charging" || name == "-h" || name == "--help" {
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return append(flags, pos...)
}

func do(method, path string, body any, expectJSON bool) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(server, "/")+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("X-Nilemu-Token", token)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if expectJSON && len(data) > 0 {
		prettyPrint(data)
	}
	return nil
}

func getJSON(path string) error {
	return do("GET", path, nil, true)
}

func postJSON(path string, body any, expectJSON bool) error {
	return do("POST", path, body, expectJSON)
}

func fetch(path string) (json.RawMessage, error) {
	req, err := http.NewRequest("GET", strings.TrimRight(server, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("X-Nilemu-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func prettyPrint(data []byte) {
	var v any
	if json.Unmarshal(data, &v) != nil {
		fmt.Println(string(data))
		return
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(out))
}

func atof(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
