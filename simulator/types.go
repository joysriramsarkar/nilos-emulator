package main

import (
	"encoding/json"
	"time"
)

// AlapNode represents a declarative UI element node in the NilUI / Alap Tree
type AlapNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"` // Column, Row, Header, TextField, Button, Text, Card, Checklist, StatusBar, Receipt
	Props    map[string]any `json:"props,omitempty"`
	Children []AlapNode     `json:"children,omitempty"`
	Rect     RECT           `json:"-"` // Calculated layout bounding box
}

// ChecklistItem represents a to-do item in a note
type ChecklistItem struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// Note represents a rich note in the notebook app
type Note struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Category  string          `json:"category"`
	Priority  string          `json:"priority"` // high, medium, low
	Color     string          `json:"color"`    // indigo, emerald, amber, rose, violet, cyan
	IsPinned  bool            `json:"isPinned"`
	IsTrashed bool            `json:"isTrashed"`
	Tags      []string        `json:"tags"`
	Checklist []ChecklistItem `json:"checklist"`
	UpdatedAt string          `json:"updatedAt"`
}

// MusicTrack represents an audio track in the NilOS Music player
type MusicTrack struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration string `json:"duration"`
	Genre    string `json:"genre"`
}

// DeviceProfile config
type DeviceProfile struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Width       int32   `json:"width"`
	Height      int32   `json:"height"`
	BezelRadius int32   `json:"bezelRadius"`
	DPR         float64 `json:"dpr"`
	HasPrinter  bool    `json:"hasPrinter"`
}

// NilHALState represents the virtual hardware status
type NilHALState struct {
	BatteryLevel    int      `json:"batteryLevel"`
	IsCharging      bool     `json:"isCharging"`
	NetworkMode     string   `json:"networkMode"`   // 5G, 4G, 3G, OFFLINE
	NetworkLatency  int      `json:"networkLatency"` // ms
	ThermalPaperOut bool     `json:"thermalPaperOut"`
	PrintedReceipts []string `json:"printedReceipts"`
	ScannerActive   bool     `json:"scannerActive"`
	LastScanBarcode string   `json:"lastScanBarcode"`
	SoundEnabled    bool     `json:"soundEnabled"`
	Volume          int      `json:"volume"` // 0-100
	Brightness      int      `json:"brightness"` // 0-100
}

// AppState holds the entire emulator runtime state
type AppState struct {
	// Profiles
	Profiles       []DeviceProfile
	CurrentProfile int

	// OS & Window State
	IsLocked      bool
	IsScreenOff   bool
	IsLandscape   bool
	ActiveAppID   string // "notes", "calc", "pos", "music", "terminal", "devtools"
	RunningApps   []string
	FPS           int
	LastFrameTime time.Time
	FrameCount    int

	// NilHAL Virtual Hardware
	HAL NilHALState

	// Live Alap UI Tree
	UITree AlapNode

	// Notes Application Data (Native Store)
	Notes            []Note
	ActiveCategory   string
	SearchQuery      string
	SelectedNoteID   string
	IsEditing        bool
	EditingNote      Note
	ActiveInputField string // "search", "title", "body", "task", "terminal"
	CursorBlink      bool
	ScrollOffsetY    int

	// Calculator App State
	CalcDisplay string
	CalcHistory string
	CalcPrevNum float64
	CalcOp      string
	CalcNewNum  bool

	// Music Player State
	Tracks         []MusicTrack
	CurrentTrack   int
	IsPlayingMusic bool
	MusicProgress  int // 0-100
	WaveformPhase  int

	// Terminal / REPL State
	TerminalLines []string
	TerminalInput string

	// DevTools / IPC Log
	IPCLogs []string

	// Status message on Dynamic Island
	StatusMessage    string
	StatusMessageExp time.Time

	// Touch ripple coordinates
	TouchX     int32
	TouchY     int32
	TouchAlpha int32
}

func (s *AppState) SetStatus(msg string) {
	s.StatusMessage = msg
	s.StatusMessageExp = time.Now().Add(3 * time.Second)
	s.AddIPCLog("DynamicIsland", "Status: "+msg)
}

func (s *AppState) AddIPCLog(channel, msg string) {
	entry := time.Now().Format("15:04:05.000") + " [" + channel + "] " + msg
	s.IPCLogs = append(s.IPCLogs, entry)
	if len(s.IPCLogs) > 120 {
		s.IPCLogs = s.IPCLogs[len(s.IPCLogs)-120:]
	}
}

func (n *AlapNode) GetStringProp(key, defVal string) string {
	if n.Props == nil {
		return defVal
	}
	if v, ok := n.Props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defVal
}

func (n *AlapNode) GetIntProp(key string, defVal int) int {
	if n.Props == nil {
		return defVal
	}
	if v, ok := n.Props[key]; ok {
		switch num := v.(type) {
		case int:
			return num
		case float64:
			return int(num)
		case json.Number:
			if i, err := num.Int64(); err == nil {
				return int(i)
			}
		}
	}
	return defVal
}

