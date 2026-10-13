package runtime

// TestReplayIsDeterministic is the deterministic-replay CI gate.
//
// It replays a fixed .nilrec tap sequence with record.Play{Instant: true} —
// the `nilctl replay --instant` path — through a fresh mock VM session and
// requires the exact patch stream of a live run. Instant replay ignores
// timestamps (no sleeping), which is what makes replayed time frozen and the
// comparison meaningful; the session also starts with a fixed seed and
// animations off, the same frozen config goldens use.
import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/record"
)

func startTapSession(t *testing.T, patches chan PatchMsg) *Session {
	t.Helper()
	app := AppInfo{ID: "com.nilang.test", Name: "Test App", Version: "1.0.0"}
	anim := false
	tr := NewPipeTransport(func(in io.Reader, out io.Writer) error {
		return RunMockVM(app, in, out)
	})
	sess, err := Start(StartOptions{
		App:       app,
		Transport: tr,
		Init: Init{
			Device:     DeviceInit{Profile: "nilphone-ultra", W: 390, H: 844, DPR: 3},
			Seed:       42,
			Animations: &anim,
		},
		Callbacks: Callbacks{
			OnPatch: func(p PatchMsg) { patches <- p },
		},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return sess
}

// sendTap delivers one tap and waits for its patch, so the op stream stays
// ordered and the test never depends on timing.
func sendTap(t *testing.T, sess *Session, patches chan PatchMsg, code string) string {
	t.Helper()
	if err := sess.SendInput(InputMsg{Kind: "tap", Code: code}); err != nil {
		t.Fatalf("tap %s: %v", code, err)
	}
	select {
	case p := <-patches:
		return string(p.Ops)
	case <-time.After(3 * time.Second):
		t.Fatalf("no patch after tap %s", code)
		return ""
	}
}

func tapRecording(t *testing.T, codes ...string) *record.Recording {
	t.Helper()
	rec := record.NewRecorder("nilphone-ultra", "com.nilang.test", "gate-taps")
	for _, code := range codes {
		// Timestamps are stamped at record time and ignored on instant
		// replay, exactly like a real capture.
		if err := func() error {
			data, err := json.Marshal(InputMsg{Kind: "tap", Code: code})
			if err != nil {
				return err
			}
			rec.Add(record.KindInput, "nilphone-ultra", "com.nilang.test", json.RawMessage(data))
			return nil
		}(); err != nil {
			t.Fatal(err)
		}
	}
	out := rec.Stop()
	return &out
}

// replayOps starts a fresh session and replays rec instantly, collecting one
// patch per input event like a live run does.
func replayOps(t *testing.T, rec *record.Recording, codes int) []string {
	t.Helper()
	patches := make(chan PatchMsg, 16)
	sess := startTapSession(t, patches)
	defer sess.Stop()

	var ops []string
	sink := func(ev record.Event) error {
		var in InputMsg
		if err := json.Unmarshal(ev.Data, &in); err != nil {
			return err
		}
		ops = append(ops, sendTap(t, sess, patches, in.Code))
		return nil
	}
	if err := record.Play(rec, record.PlaybackOptions{Instant: true}, sink); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(ops) != codes {
		t.Fatalf("replay produced %d op batches, want %d", len(ops), codes)
	}
	return ops
}

func liveOps(t *testing.T, codes ...string) []string {
	t.Helper()
	patches := make(chan PatchMsg, 16)
	sess := startTapSession(t, patches)
	defer sess.Stop()

	var ops []string
	for _, code := range codes {
		ops = append(ops, sendTap(t, sess, patches, code))
	}
	return ops
}

func TestReplayIsDeterministic(t *testing.T) {
	taps := []string{"h:1", "h:1", "h:1"}

	live := liveOps(t, taps...)
	rec := tapRecording(t, taps...)

	replay1 := replayOps(t, rec, len(taps))
	replay2 := replayOps(t, rec, len(taps))

	eq := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	if !eq(live, replay1) {
		t.Fatalf("replay diverged from live run:\n live: %q\nreplay: %q", live, replay1)
	}
	if !eq(replay1, replay2) {
		t.Fatalf("replay is not deterministic:\nfirst:  %q\nsecond: %q", replay1, replay2)
	}
	for _, ops := range [][]string{live, replay1, replay2} {
		for _, batch := range ops {
			if !strings.Contains(batch, "counter") {
				t.Fatalf("expected counter patches, got %q", batch)
			}
		}
	}
}
