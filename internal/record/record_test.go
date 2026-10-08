package record

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestRecorderSaveLoad(t *testing.T) {
	r := NewRecorder("dev-1", "com.example.app", "test-rec")
	r.Add(KindInput, "dev-1", "com.example.app", map[string]any{"kind": "tap", "x": 10, "y": 20})
	r.Add(KindHAL, "dev-1", "com.example.app", map[string]any{"dev": "printer", "op": "print"})

	if r.Len() != 2 {
		t.Fatalf("expected 2 events, got %d", r.Len())
	}

	path := filepath.Join(t.TempDir(), "test.nilrec")
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	rec, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Name != "test-rec" || len(rec.Events) != 2 {
		t.Fatalf("unexpected recording: %+v", rec)
	}
	if rec.Events[0].Kind != KindInput || rec.Events[1].Kind != KindHAL {
		t.Fatalf("unexpected event kinds: %+v", rec.Events)
	}
}

func TestPlayOrderAndInstant(t *testing.T) {
	rec := &Recording{Events: []Event{
		{Ts: 0, Kind: KindInput, Data: json.RawMessage(`{"n":1}`)},
		{Ts: 5000, Kind: KindInput, Data: json.RawMessage(`{"n":2}`)},
	}}
	var order []int
	start := time.Now()
	err := Play(rec, PlaybackOptions{Instant: true}, func(ev Event) error {
		var m map[string]int
		json.Unmarshal(ev.Data, &m)
		order = append(order, m["n"])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("unexpected order: %v", order)
	}
	if time.Since(start) > time.Second {
		t.Fatal("instant replay should not wait for timestamps")
	}
}
