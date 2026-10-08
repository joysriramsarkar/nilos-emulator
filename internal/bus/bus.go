// Package bus implements NilBus, the inter-app and app-to-service message
// router. It supports topic pub/sub and request/response, and keeps a rolling
// log that feeds the DevTools IPC monitor.
package bus

import (
	"encoding/json"
	"sync"
	"time"
)

// Message is one NilBus message.
type Message struct {
	ID      uint64          `json:"id"`
	Kind    string          `json:"kind"` // publish | request | response
	Topic   string          `json:"topic"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Ts      int64           `json:"ts"`
}

// Bus is the message router.
type Bus struct {
	mu      sync.RWMutex
	subs    map[string]map[int]chan Message
	nextSub int
	log     []Message
	logCap  int
	seq     uint64
}

// New creates a bus retaining the last logCap messages.
func New(logCap int) *Bus {
	if logCap <= 0 {
		logCap = 1000
	}
	return &Bus{subs: map[string]map[int]chan Message{}, logCap: logCap}
}

// Publish sends a message to all subscribers of topic.
func (b *Bus) Publish(topic, from string, payload json.RawMessage) Message {
	b.mu.Lock()
	b.seq++
	m := Message{ID: b.seq, Kind: "publish", Topic: topic, From: from, Payload: payload, Ts: time.Now().UnixMilli()}
	b.appendLocked(m)
	targets := b.targetsLocked(topic)
	b.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- m:
		default:
		}
	}
	return m
}

// Request publishes a request and returns its message (responses are
// delivered to subscribers of topic+".response").
func (b *Bus) Request(topic, from, to string, payload json.RawMessage) Message {
	b.mu.Lock()
	b.seq++
	m := Message{ID: b.seq, Kind: "request", Topic: topic, From: from, To: to, Payload: payload, Ts: time.Now().UnixMilli()}
	b.appendLocked(m)
	targets := b.targetsLocked(topic)
	b.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- m:
		default:
		}
	}
	return m
}

// Respond publishes a response message.
func (b *Bus) Respond(topic, from, to string, payload json.RawMessage) Message {
	b.mu.Lock()
	b.seq++
	m := Message{ID: b.seq, Kind: "response", Topic: topic, From: from, To: to, Payload: payload, Ts: time.Now().UnixMilli()}
	b.appendLocked(m)
	targets := b.targetsLocked(topic)
	b.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- m:
		default:
		}
	}
	return m
}

// Subscribe returns a channel of messages for topic and a cancel func.
func (b *Bus) Subscribe(topic string, buffer int) (<-chan Message, func()) {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan Message, buffer)
	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = map[int]chan Message{}
	}
	id := b.nextSub
	b.nextSub++
	b.subs[topic][id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		if group, ok := b.subs[topic]; ok {
			if existing, ok := group[id]; ok {
				delete(group, id)
				close(existing)
			}
			if len(group) == 0 {
				delete(b.subs, topic)
			}
		}
		b.mu.Unlock()
	}
	return ch, cancel
}

// Log returns the last n messages (all if n <= 0).
func (b *Bus) Log(n int) []Message {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if n <= 0 || n > len(b.log) {
		n = len(b.log)
	}
	out := make([]Message, n)
	copy(out, b.log[len(b.log)-n:])
	return out
}

func (b *Bus) appendLocked(m Message) {
	b.log = append(b.log, m)
	if len(b.log) > b.logCap {
		b.log = b.log[len(b.log)-b.logCap:]
	}
}

func (b *Bus) targetsLocked(topic string) []chan Message {
	var out []chan Message
	for _, ch := range b.subs[topic] {
		out = append(out, ch)
	}
	// wildcard subscribers
	for _, ch := range b.subs["*"] {
		out = append(out, ch)
	}
	return out
}
