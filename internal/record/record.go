// Package record captures and replays input and HAL events so a bug report can
// be reproduced exactly. A recording is stored as a .nilrec JSON file.
package record

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Kind classifies a recorded event.
type Kind string

const (
	KindInput     Kind = "input"
	KindIME       Kind = "ime"
	KindHAL       Kind = "hal"
	KindLifecycle Kind = "lifecycle"
	KindConfig    Kind = "config"
)

// Event is one timestamped action. Ts is milliseconds since recording start.
type Event struct {
	Ts     int64           `json:"ts"`
	Kind   Kind            `json:"kind"`
	Device string          `json:"device,omitempty"`
	App    string          `json:"app,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// Recording is a saved session capture.
type Recording struct {
	Name      string  `json:"name"`
	CreatedAt int64   `json:"created_at"`
	Device    string  `json:"device"`
	App       string  `json:"app"`
	Events    []Event `json:"events"`
}

// Recorder accumulates events for one app session.
type Recorder struct {
	mu      sync.Mutex
	rec     Recording
	started time.Time
}

// NewRecorder starts a recording for a device/app.
func NewRecorder(device, app, name string) *Recorder {
	return &Recorder{
		started: time.Now(),
		rec: Recording{
			Name:      name,
			CreatedAt: time.Now().UnixMilli(),
			Device:    device,
			App:       app,
		},
	}
}

// Add records one event.
func (r *Recorder) Add(kind Kind, device, app string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec.Events = append(r.rec.Events, Event{
		Ts:     time.Since(r.started).Milliseconds(),
		Kind:   kind,
		Device: device,
		App:    app,
		Data:   data,
	})
}

// Snapshot returns a copy of the recording so far.
func (r *Recorder) Snapshot() Recording {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.rec
	out.Events = append([]Event(nil), r.rec.Events...)
	return out
}

// Stop finalises and returns the recording.
func (r *Recorder) Stop() Recording { return r.Snapshot() }

// Len returns the number of recorded events.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.rec.Events)
}

// Save writes the recording to a .nilrec file.
func (r *Recorder) Save(path string) error {
	rec := r.Snapshot()
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load reads a .nilrec file.
func Load(path string) (*Recording, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// PlaybackOptions controls replay timing.
type PlaybackOptions struct {
	// Speed multiplies time (2 = twice as fast). <=0 means 1x.
	Speed float64
	// Instant ignores timestamps and replays as fast as possible.
	Instant bool
}

// Play replays the recording through sink, honouring event timing.
func Play(rec *Recording, opts PlaybackOptions, sink func(Event) error) error {
	speed := opts.Speed
	if speed <= 0 {
		speed = 1
	}
	var prev int64
	for _, ev := range rec.Events {
		if !opts.Instant {
			delay := ev.Ts - prev
			prev = ev.Ts
			if delay > 0 {
				time.Sleep(time.Duration(float64(delay)/speed) * time.Millisecond)
			}
		}
		if err := sink(ev); err != nil {
			return err
		}
	}
	return nil
}
