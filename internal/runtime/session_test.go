package runtime

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSessionHandshakeFrameAndPatch(t *testing.T) {
	app := AppInfo{ID: "com.nilang.test", Name: "Test App", Version: "1.0.0"}
	tr := NewPipeTransport(func(in io.Reader, out io.Writer) error {
		return RunMockVM(app, in, out)
	})
	frames := make(chan FrameMsg, 16)
	patches := make(chan PatchMsg, 16)
	states := make(chan StateMsg, 4)

	sess, err := Start(StartOptions{
		App:       app,
		Transport: tr,
		Init:      Init{Device: DeviceInit{Profile: "nilphone-ultra", W: 390, H: 844, DPR: 3}},
		Callbacks: Callbacks{
			OnFrame: func(f FrameMsg) { frames <- f },
			OnPatch: func(p PatchMsg) { patches <- p },
			OnState: func(s StateMsg) { states <- s },
		},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer sess.Stop()

	select {
	case f := <-frames:
		if len(f.Tree) == 0 {
			t.Fatal("empty tree")
		}
		if !strings.Contains(string(f.Tree), "header") {
			t.Fatalf("tree missing header: %s", f.Tree)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no initial frame")
	}

	// h:1 is the increment handler for the default counter app.
	if err := sess.SendInput(InputMsg{Kind: "tap", Code: "h:1"}); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-patches:
		if !strings.Contains(string(p.Ops), "counter") {
			t.Fatalf("unexpected patch: %s", p.Ops)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no patch after tap")
	}

	if err := sess.SendLifecycle(LifecyclePaused); err != nil {
		t.Fatal(err)
	}
	select {
	case st := <-states:
		if len(st.Blob) == 0 {
			t.Fatal("empty saved state")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no state on pause")
	}
}

func TestHandshakeTimeout(t *testing.T) {
	tr := NewPipeTransport(func(in io.Reader, out io.Writer) error {
		// Never send hello.
		time.Sleep(2 * time.Second)
		return nil
	})
	_, err := Start(StartOptions{
		App:              AppInfo{ID: "x"},
		Transport:        tr,
		HandshakeTimeout: 150 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected handshake timeout")
	}
}

func TestProtocolMismatch(t *testing.T) {
	tr := NewPipeTransport(func(in io.Reader, out io.Writer) error {
		b, _ := json.Marshal(Hello{Type: "hello", Proto: 999, Runtime: "fake"})
		out.Write(append(b, '\n'))
		time.Sleep(2 * time.Second)
		return nil
	})
	_, err := Start(StartOptions{
		App:              AppInfo{ID: "x"},
		Transport:        tr,
		HandshakeTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected protocol mismatch")
	}
}

func TestHALCallRoundTrip(t *testing.T) {
	app := AppInfo{ID: "com.nilang.test", Name: "Test"}
	tr := NewPipeTransport(func(in io.Reader, out io.Writer) error {
		return RunMockVM(app, in, out)
	})
	called := make(chan string, 4)
	sess, err := Start(StartOptions{
		App:       app,
		Transport: tr,
		Init:      Init{Device: DeviceInit{W: 390, H: 844}},
		HAL: func(dev, op string, args json.RawMessage) (any, error) {
			called <- dev + "." + op
			return 77, nil
		},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer sess.Stop()

	select {
	case c := <-called:
		if c != "battery.level" {
			t.Fatalf("unexpected HAL call: %s", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("HAL call never arrived")
	}
}
