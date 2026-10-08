package hal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joysriramsarkar/nilos-emulator/internal/vfs"
)

func TestESCPostRoundTrip(t *testing.T) {
	bytes := EncodeReceipt([]string{"NILPOS RECEIPT", "চাল ২ x 3200 = 6400", "VAT 5% = 320", "TOTAL = 6720"})
	doc := DecodeESCPS(bytes)
	if len(doc.Lines) == 0 {
		t.Fatal("no lines decoded")
	}
	if !doc.Cut {
		t.Fatal("expected cut command")
	}
	var all strings.Builder
	for _, l := range doc.Lines {
		all.WriteString(l.Text)
		all.WriteString("\n")
	}
	for _, want := range []string{"NILPOS RECEIPT", "TOTAL = 6720", "চাল"} {
		if !strings.Contains(all.String(), want) {
			t.Errorf("decoded receipt missing %q:\n%s", want, all.String())
		}
	}
	if doc.Lines[0].Align != "center" || !doc.Lines[0].Bold {
		t.Errorf("header should be centered+bold: %+v", doc.Lines[0])
	}
}

func TestBatteryAndNetwork(t *testing.T) {
	d := NewDispatcher(Config{DeviceID: "dev1"})
	if _, err := d.SetBattery(15, false); err != nil {
		t.Fatal(err)
	}
	state := d.Snapshot()
	if state.Battery != 15 || state.Charging {
		t.Fatalf("unexpected battery state: %+v", state)
	}
	if _, err := d.SetNetwork("3g"); err != nil {
		t.Fatal(err)
	}
	state = d.Snapshot()
	if state.NetworkMode != "3g" || !state.Online || state.LatencyMs != 300 {
		t.Fatalf("unexpected network state: %+v", state)
	}
	if _, err := d.SetNetwork("offline"); err != nil {
		t.Fatal(err)
	}
	if d.Snapshot().Online {
		t.Fatal("expected offline")
	}
}

func TestFaultInjection(t *testing.T) {
	d := NewDispatcher(Config{DeviceID: "dev1"})
	d.InjectFault(FaultRule{Dev: "printer", Op: "print", Code: "paper_out", Times: 1})

	if _, err := d.Call("printer", "print", json.RawMessage(`{"lines":["x"]}`)); err == nil {
		t.Fatal("expected injected fault")
	}
	if _, err := d.Call("printer", "print", json.RawMessage(`{"lines":["x"]}`)); err != nil {
		t.Fatalf("second print should succeed: %v", err)
	}
}

func TestStorageIsolation(t *testing.T) {
	root := t.TempDir()
	fsys, err := vfs.New(root)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(Config{DeviceID: "dev1", VFS: fsys})
	writeArgs, _ := json.Marshal(map[string]any{"path": "notes/data.json", "data": "hello"})
	if _, err := d.Call("storage", "write", writeArgs); err != nil {
		t.Fatal(err)
	}
	readArgs, _ := json.Marshal(map[string]any{"path": "notes/data.json"})
	res, err := d.Call("storage", "read", readArgs)
	if err != nil {
		t.Fatal(err)
	}
	m := res.(map[string]any)
	if m["data"] != "hello" {
		t.Fatalf("unexpected read result: %+v", m)
	}
	escapeArgs, _ := json.Marshal(map[string]any{"path": "../../escape.txt"})
	if _, err := d.Call("storage", "write", escapeArgs); err == nil {
		t.Fatal("expected escape to be rejected")
	}
}

func TestPermissionEnforced(t *testing.T) {
	d := NewDispatcher(Config{DeviceID: "dev1", Permit: func(device, op string) error {
		return vfs.ErrEscape // simulate denial
	}})
	if _, err := d.Call("printer", "print", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected permission error")
	}
}
