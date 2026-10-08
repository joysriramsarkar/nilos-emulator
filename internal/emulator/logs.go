package emulator

import "sync"

// LogEntry is a single logcat-style record.
type LogEntry struct {
	Timestamp int64  `json:"ts"`
	DeviceID  string `json:"device_id,omitempty"`
	AppID     string `json:"app_id,omitempty"`
	Level     string `json:"level"`
	Tag       string `json:"tag"`
	Message   string `json:"message"`
}

// LogHub is a bounded in-memory log ring buffer with fan-out subscribers.
type LogHub struct {
	mu      sync.RWMutex
	buf     []LogEntry
	cap     int
	subs    map[int]chan LogEntry
	nextSub int
}

// NewLogHub creates a log hub holding up to capacity entries.
func NewLogHub(capacity int) *LogHub {
	if capacity <= 0 {
		capacity = 10000
	}
	return &LogHub{cap: capacity, subs: map[int]chan LogEntry{}}
}

// Emit records an entry and fans it out to subscribers without blocking.
func (h *LogHub) Emit(e LogEntry) {
	if e.Timestamp == 0 {
		e.Timestamp = nowMillis()
	}
	h.mu.Lock()
	h.buf = append(h.buf, e)
	if len(h.buf) > h.cap {
		h.buf = h.buf[len(h.buf)-h.cap:]
	}
	subs := make([]chan LogEntry, 0, len(h.subs))
	for _, ch := range h.subs {
		subs = append(subs, ch)
	}
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// Recent returns the last n entries (all, if n <= 0).
func (h *LogHub) Recent(n int) []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if n <= 0 || n > len(h.buf) {
		n = len(h.buf)
	}
	out := make([]LogEntry, n)
	copy(out, h.buf[len(h.buf)-n:])
	return out
}

// Subscribe returns a channel of future entries and a cancel function.
func (h *LogHub) Subscribe(buffer int) (<-chan LogEntry, func()) {
	if buffer <= 0 {
		buffer = 256
	}
	ch := make(chan LogEntry, buffer)
	h.mu.Lock()
	id := h.nextSub
	h.nextSub++
	h.subs[id] = ch
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if existing, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(existing)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}
