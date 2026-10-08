package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrHandshakeTimeout is returned when the VM does not send a hello in time.
var ErrHandshakeTimeout = errors.New("runtime: handshake timed out")

// ErrProtocolMismatch is returned when the VM speaks an incompatible protocol.
var ErrProtocolMismatch = errors.New("runtime: protocol version mismatch")

// ErrStopped is returned when sending on a stopped session.
var ErrStopped = errors.New("runtime: session stopped")

// AppInfo identifies the app being hosted.
type AppInfo struct {
	ID          string
	Name        string
	Version     string
	Entry       string
	Dir         string
	Permissions []string
}

// HalFunc handles a hal.call from the VM.
type HalFunc func(dev, op string, args json.RawMessage) (any, error)

// Callbacks receives asynchronous VM events.
type Callbacks struct {
	OnFrame func(FrameMsg)
	OnPatch func(PatchMsg)
	OnLog   func(LogMsg)
	OnTrace func(TraceMsg)
	OnState func(StateMsg)
	OnExit  func(ExitMsg)
	OnError func(ErrorMsg)
	OnBus   func(BusMsg)
}

// StartOptions configures a session.
type StartOptions struct {
	App              AppInfo
	Transport        Transport
	Init             Init
	HAL              HalFunc
	Callbacks        Callbacks
	HandshakeTimeout time.Duration
}

type outMsg struct {
	data []byte
	kind string
}

// Session is a live connection to one NilLang app worker.
type Session struct {
	ID      string
	App     AppInfo
	Hello   Hello
	tr      Transport
	scanner *bufio.Scanner

	wmu    sync.Mutex
	writer interface {
		Write([]byte) (int, error)
	}
	out chan outMsg

	mu      sync.Mutex
	state   string
	lastSeq uint64
	tree    json.RawMessage
	patches uint64
	frames  uint64
	saved   json.RawMessage

	hal HalFunc
	cb  Callbacks

	helloCh  chan Hello
	done     chan struct{}
	closed   chan struct{}
	exitOnce sync.Once
	exitErr  error
}

// Start wires up a session, performs the NilHost handshake and returns it.
func Start(opts StartOptions) (*Session, error) {
	timeout := opts.HandshakeTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	s := &Session{
		ID:      fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		App:     opts.App,
		tr:      opts.Transport,
		hal:     opts.HAL,
		cb:      opts.Callbacks,
		out:     make(chan outMsg, 256),
		helloCh: make(chan Hello, 1),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
		state:   string(LifecycleStarted),
	}
	s.writer = opts.Transport.Writer()
	s.scanner = bufio.NewScanner(opts.Transport.Reader())
	s.scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	go s.readLoop()
	go s.writeLoop()

	select {
	case h := <-s.helloCh:
		s.Hello = h
		if h.Proto != ProtoVersion {
			s.Stop()
			return nil, fmt.Errorf("%w: host=%d vm=%d", ErrProtocolMismatch, ProtoVersion, h.Proto)
		}
	case <-time.After(timeout):
		s.Stop()
		return nil, ErrHandshakeTimeout
	case <-s.closed:
		return nil, fmt.Errorf("runtime: vm exited during handshake")
	}

	initMsg := opts.Init
	initMsg.Type = "init"
	initMsg.Proto = ProtoVersion
	if err := s.Send(initMsg); err != nil {
		s.Stop()
		return nil, err
	}
	return s, nil
}

func (s *Session) readLoop() {
	defer close(s.done)
	for s.scanner.Scan() {
		line := append([]byte(nil), s.scanner.Bytes()...)
		if !validLine(line) {
			s.emitLog(LogMsg{Level: "debug", Tag: "vm", Msg: string(line), Ts: time.Now().UnixMilli()})
			continue
		}
		var env Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}
		s.dispatch(env.Type, line)
	}
	if err := s.scanner.Err(); err != nil {
		s.emitLog(LogMsg{Level: "error", Tag: "protocol", Msg: "read error: " + err.Error()})
	}
	s.finish(ExitMsg{Code: 0, Reason: "stream closed"})
}

func (s *Session) dispatch(t string, line []byte) {
	switch t {
	case "hello":
		var h Hello
		if json.Unmarshal(line, &h) == nil {
			select {
			case s.helloCh <- h:
			default:
			}
		}
	case "frame":
		var f FrameMsg
		if json.Unmarshal(line, &f) == nil {
			s.mu.Lock()
			s.tree = f.Tree
			s.frames++
			if f.Seq > s.lastSeq {
				s.lastSeq = f.Seq
			}
			s.mu.Unlock()
			if s.cb.OnFrame != nil {
				s.cb.OnFrame(f)
			}
		}
	case "patch":
		var p PatchMsg
		if json.Unmarshal(line, &p) == nil {
			s.mu.Lock()
			s.patches++
			s.mu.Unlock()
			if s.cb.OnPatch != nil {
				s.cb.OnPatch(p)
			}
		}
	case "hal.call":
		var c HalCallMsg
		if json.Unmarshal(line, &c) == nil {
			go s.handleHalCall(c)
		}
	case "log":
		var l LogMsg
		if json.Unmarshal(line, &l) == nil {
			s.emitLog(l)
		}
	case "trace":
		var tr TraceMsg
		if json.Unmarshal(line, &tr) == nil && s.cb.OnTrace != nil {
			s.cb.OnTrace(tr)
		}
	case "state":
		var st StateMsg
		if json.Unmarshal(line, &st) == nil {
			s.mu.Lock()
			s.saved = st.Blob
			s.mu.Unlock()
			if s.cb.OnState != nil {
				s.cb.OnState(st)
			}
		}
	case "bus":
		var b BusMsg
		if json.Unmarshal(line, &b) == nil && s.cb.OnBus != nil {
			s.cb.OnBus(b)
		}
	case "error":
		var e ErrorMsg
		if json.Unmarshal(line, &e) == nil {
			if s.cb.OnError != nil {
				s.cb.OnError(e)
			}
			s.emitLog(LogMsg{Level: "error", Tag: "runtime", Msg: e.Msg})
		}
	case "exit":
		var e ExitMsg
		if json.Unmarshal(line, &e) == nil {
			s.finish(e)
			_ = s.tr.Close()
		}
	}
}

func (s *Session) handleHalCall(c HalCallMsg) {
	var result HalResultMsg
	result.Type = "hal.result"
	result.ID = c.ID
	if s.hal == nil {
		result.OK = false
		result.Error = "no HAL handler"
	} else {
		data, err := s.hal(c.Dev, c.Op, c.Args)
		if err != nil {
			result.OK = false
			result.Error = err.Error()
		} else {
			result.OK = true
			if data != nil {
				if b, mErr := json.Marshal(data); mErr == nil {
					result.Data = b
				}
			}
		}
	}
	_ = s.Send(result)
}

func (s *Session) emitLog(l LogMsg) {
	if l.Ts == 0 {
		l.Ts = time.Now().UnixMilli()
	}
	if s.cb.OnLog != nil {
		s.cb.OnLog(l)
	}
}

func (s *Session) writeLoop() {
	for {
		select {
		case m, ok := <-s.out:
			if !ok {
				return
			}
			s.wmu.Lock()
			_, err := s.writer.Write(m.data)
			s.wmu.Unlock()
			if err != nil {
				s.finish(ExitMsg{Code: 1, Reason: "write error: " + err.Error()})
				return
			}
		case <-s.closed:
			return
		}
	}
}

// Send serialises v as one protocol line.
func (s *Session) Send(v any) error {
	select {
	case <-s.closed:
		return ErrStopped
	default:
	}
	data, err := encodeLine(v)
	if err != nil {
		return err
	}
	return s.enqueue(outMsg{data: data, kind: msgKind(v)})
}

func (s *Session) enqueue(m outMsg) error {
	select {
	case s.out <- m:
		return nil
	default:
	}
	if m.kind == "move" {
		return nil // drop coalesced moves under pressure
	}
	// Make room by dropping one queued message, then retry briefly.
	select {
	case <-s.out:
	default:
	}
	select {
	case s.out <- m:
		return nil
	case <-time.After(2 * time.Second):
		return errors.New("runtime: backpressure, message dropped")
	case <-s.closed:
		return ErrStopped
	}
}

// SendInput sends a raw input event.
func (s *Session) SendInput(in InputMsg) error {
	in.Type = "input"
	if in.Ts == 0 {
		in.Ts = time.Now().UnixMilli()
	}
	return s.Send(in)
}

// SendLifecycle sends a lifecycle transition.
func (s *Session) SendLifecycle(state string) error {
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
	return s.Send(LifecycleMsg{Type: "lifecycle", State: state})
}

// SendConfig sends a configuration change.
func (s *Session) SendConfig(c ConfigMsg) error {
	c.Type = "config"
	return s.Send(c)
}

// SendVsync sends a frame credit.
func (s *Session) SendVsync() error {
	return s.Send(VsyncMsg{Type: "vsync", Ts: time.Now().UnixMilli()})
}

// SendIME sends an IME command.
func (s *Session) SendIME(m ImeMsg) error {
	m.Type = "ime"
	return s.Send(m)
}

// SendReload requests a hot restart/reload.
func (s *Session) SendReload(mode string) error {
	return s.Send(ReloadMsg{Type: "reload", Mode: mode})
}

// SendDebug forwards a debugger command.
func (s *Session) SendDebug(cmd string, args map[string]any) error {
	return s.Send(DebugMsg{Type: "debug", Cmd: cmd, Args: args})
}

// State returns the current lifecycle state string.
func (s *Session) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// LastTree returns the most recent full UI tree.
func (s *Session) LastTree() json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tree
}

// SavedState returns the last state blob the VM sent.
func (s *Session) SavedState() json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved
}

// Stats returns frame/patch counters.
func (s *Session) Stats() (frames, patches uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames, s.patches
}

// Pid returns the worker process id (0 for in-process VMs).
func (s *Session) Pid() int { return s.tr.Pid() }

// Done returns a channel closed when the VM stream ends.
func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) finish(e ExitMsg) {
	s.exitOnce.Do(func() {
		s.mu.Lock()
		s.state = LifecycleDestroyed
		s.mu.Unlock()
		close(s.closed)
		if s.cb.OnExit != nil {
			s.cb.OnExit(e)
		}
	})
}

// Stop terminates the session and waits briefly for the worker to exit.
func (s *Session) Stop() error {
	_ = s.SendLifecycle(string(LifecycleStopped))
	err := s.tr.Close()
	select {
	case <-s.closed:
	case <-time.After(2 * time.Second):
	}
	return err
}

// Wait blocks until the session ends.
func (s *Session) Wait() { <-s.done }

func msgKind(v any) string {
	switch m := v.(type) {
	case InputMsg:
		if m.Phase == "move" {
			return "move"
		}
	case *InputMsg:
		if m.Phase == "move" {
			return "move"
		}
	}
	return "other"
}

// Lifecycle constants duplicated as plain strings to keep the runtime package
// independent of the emulator package.
const (
	LifecycleCreated   = "created"
	LifecycleStarted   = "started"
	LifecycleResumed   = "resumed"
	LifecyclePaused    = "paused"
	LifecycleStopped   = "stopped"
	LifecycleDestroyed = "destroyed"
)
