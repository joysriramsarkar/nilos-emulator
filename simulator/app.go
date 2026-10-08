package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NewAppState() *AppState {
	s := &AppState{
		Profiles: []DeviceProfile{
			{ID: "nilphone", Name: "NilPhone 16 Pro Max", Width: 370, Height: 780, BezelRadius: 50, DPR: 3.0, HasPrinter: false},
			{ID: "nilpad", Name: "NilPad Pro Tablet", Width: 540, Height: 780, BezelRadius: 36, DPR: 2.0, HasPrinter: false},
			{ID: "nilpos", Name: "NilPOS Retail Terminal", Width: 420, Height: 780, BezelRadius: 28, DPR: 2.0, HasPrinter: true},
		},
		CurrentProfile: 0,
		IsLocked:       false,
		IsScreenOff:    false,
		IsLandscape:    false,
		ActiveAppID:    "notes",
		RunningApps:    []string{"notes", "calc", "pos", "music", "terminal"},
		FPS:            60,
		LastFrameTime:  time.Now(),
		FrameCount:     0,
		HAL: NilHALState{
			BatteryLevel:    96,
			IsCharging:      true,
			NetworkMode:     "5G",
			NetworkLatency:  4,
			ThermalPaperOut: false,
			PrintedReceipts: []string{},
			ScannerActive:   false,
			LastScanBarcode: "8901030012345",
			SoundEnabled:    true,
			Volume:          85,
			Brightness:      100,
		},
		Notes: []Note{
			{
				ID:        "note-1",
				Title:     "নীলাং নোটবুক ও অনূরণ ওএস আর্কিটেকচার",
				Body:      "সম্পূর্ণ দেশীয় নীলাং (NilLang) ও আলাপ (Alap) ডিক্লারেটিভ UI ফ্রেমওয়ার্কে তৈরি রিয়েল-টাইম রিচ নোটবুক অ্যাপ্লিকেশন। এটি x86_64 ও ARM64 প্ল্যাটফর্মে সম্পূর্ণ কার্যকর।",
				Category:  "important",
				Priority:  "high",
				Color:     "indigo",
				IsPinned:  true,
				IsTrashed: false,
				Tags:      []string{"nilang", "onuron", "architecture"},
				Checklist: []ChecklistItem{
					{ID: "c1", Text: "স্মার্ট ক্যাটাগরি ও ফিল্টারিং সিস্টেম", Completed: true},
					{ID: "c2", Text: "লাইভ ইন্টারঅ্যাক্টিভ চেকলিস্ট ও প্রোগ্রেস বার", Completed: true},
					{ID: "c3", Text: "১০০% পিওর নেটিভ উইন্ডোজ ফোন এমুলেটর", Completed: true},
				},
				UpdatedAt: "আজ ০৯:৩০",
			},
			{
				ID:        "note-2",
				Title:     "আজকের কাজের পরিকল্পনা ও টাস্ক লিস্ট",
				Body:      "সফটবাস আইপিসি এবং ডিক্লারেটিভ UI টেস্ট বেঞ্চমার্ক শেষ করতে হবে। সন্ধ্যা ৬টার মধ্যে রিগ্রেশন টেস্ট সম্পন্ন করা আবশ্যক।",
				Category:  "work",
				Priority:  "high",
				Color:     "emerald",
				IsPinned:  true,
				IsTrashed: false,
				Tags:      []string{"work", "tasks", "deadline"},
				Checklist: []ChecklistItem{
					{ID: "c4", Text: "নীলাং কম্পাইলার অপটিমাইজেশন ভেরিফাই করা", Completed: true},
					{ID: "c5", Text: "অনুরণ মাইক্রোকার্নেল ডিভাইস ড্রাইভার ইন্টিগ্রেশন", Completed: false},
					{ID: "c6", Text: "ইউজার ইন্টারফেসে টাচ হ্যাপটিক্স রেসপন্স টেস্ট", Completed: false},
				},
				UpdatedAt: "আজ ০৮:১৫",
			},
			{
				ID:        "note-3",
				Title:     "সৃজনশীল চিন্তা ও নতুন প্রোজেক্ট আইডিয়া",
				Body:      "ভবিষ্যতে নীল ওএস-এর জন্য একটি বাংলা ভয়েস অ্যাসিস্ট্যান্ট এবং লাইভ ওয়েভফর্ম যুক্ত মিউজিক প্লেয়ার তৈরি করার পরিকল্পনা।",
				Category:  "ideas",
				Priority:  "medium",
				Color:     "amber",
				IsPinned:  false,
				IsTrashed: false,
				Tags:      []string{"ideas", "ai", "voice"},
				Checklist: []ChecklistItem{
					{ID: "c7", Text: "বাংলা ফোনেটিক সাউন্ড মডেল তৈরি", Completed: false},
				},
				UpdatedAt: "গতকাল ১১:৪৫",
			},
			{
				ID:        "note-4",
				Title:     "প্রিয় বই ও উদ্ধৃতি সংগ্রহ",
				Body:      "\"কল্পনা জ্ঞানের চেয়েও শক্তিশালী। জ্ঞান সীমিত, কিন্তু কল্পনা পুরো বিশ্বকে পরিবেষ্টন করে রাখে।\" — আলবার্ট আইনস্টাইন।",
				Category:  "personal",
				Priority:  "low",
				Color:     "violet",
				IsPinned:  false,
				IsTrashed: false,
				Tags:      []string{"quotes", "personal", "books"},
				Checklist: []ChecklistItem{},
				UpdatedAt: "৬ অক্টোবর",
			},
		},
		ActiveCategory:   "all",
		SearchQuery:      "",
		SelectedNoteID:   "",
		IsEditing:        false,
		ActiveInputField: "search",
		CursorBlink:      true,
		ScrollOffsetY:    0,
		CalcDisplay:      "0",
		CalcHistory:      "",
		CalcPrevNum:      0,
		CalcOp:           "",
		CalcNewNum:       true,
		Tracks: []MusicTrack{
			{ID: "t1", Title: "নীল আকাশ ও সুরের ধারা", Artist: "অনুরণ সাউন্ড স্টুডিও", Duration: "০৩:৪৫", Genre: "অ্যাকোস্টিক"},
			{ID: "t2", Title: "মেঘের গান ও বৃষ্টি", Artist: "নীলাং মেলোডি ব্যান্ড", Duration: "০৪:১২", Genre: "ফিউশন"},
			{ID: "t3", Title: "সোনার বাংলা নতুন দিগন্ত", Artist: "ঐক্য তানবাদন", Duration: "০২:৫৮", Genre: "দেশাত্মবোধক"},
		},
		CurrentTrack:   0,
		IsPlayingMusic: false,
		MusicProgress:  32,
		WaveformPhase:  0,
		TerminalLines: []string{
			"NilOS v2.0 (x86_64-onuron-elf) Microkernel",
			"Welcome to NilLang Interactive Shell REPL",
			"Type 'help', 'apps', 'mem', 'ps' or NilLang expressions.",
			"nil> ",
		},
		TerminalInput:    "",
		IPCLogs:          []string{},
		StatusMessage:    "NilPhone 16 Pro Max Native Ready",
		StatusMessageExp: time.Now().Add(4 * time.Second),
	}

	s.AddIPCLog("System", "NilOS v2.0 Microkernel booted (arch: x86_64)")
	s.AddIPCLog("NilHAL", "Display initialized: 120Hz Super Retina AMOLED")
	s.AddIPCLog("NilBus", "SoftBus IPC broker listening on channel 0..11")
	return s
}

func (s *AppState) GetActiveProfile() DeviceProfile {
	if s.CurrentProfile >= 0 && s.CurrentProfile < len(s.Profiles) {
		return s.Profiles[s.CurrentProfile]
	}
	return s.Profiles[0]
}

func (s *AppState) SwitchProfile(idx int) {
	if idx >= 0 && idx < len(s.Profiles) {
		s.CurrentProfile = idx
		p := s.Profiles[idx]
		s.SetStatus("প্রোফাইল: " + p.Name)
		s.AddIPCLog("DeviceManager", "Switched profile to: "+p.ID)
		PlayClickTone()
	}
}

func (s *AppState) FilteredNotes() []Note {
	var list []Note
	q := strings.ToLower(strings.TrimSpace(s.SearchQuery))

	for _, n := range s.Notes {
		if s.ActiveCategory == "trash" {
			if !n.IsTrashed {
				continue
			}
		} else {
			if n.IsTrashed {
				continue
			}
			if s.ActiveCategory != "all" && n.Category != s.ActiveCategory {
				continue
			}
		}

		if q != "" {
			matchTitle := strings.Contains(strings.ToLower(n.Title), q)
			matchBody := strings.Contains(strings.ToLower(n.Body), q)
			matchTag := false
			for _, t := range n.Tags {
				if strings.Contains(strings.ToLower(t), q) {
					matchTag = true
					break
				}
			}
			if !matchTitle && !matchBody && !matchTag {
				continue
			}
		}

		list = append(list, n)
	}

	var pinned, normal []Note
	for _, n := range list {
		if n.IsPinned {
			pinned = append(pinned, n)
		} else {
			normal = append(normal, n)
		}
	}
	return append(pinned, normal...)
}

func (s *AppState) HandleCalcButton(label string) {
	PlayClickTone()
	switch label {
	case "AC":
		s.CalcDisplay = "0"
		s.CalcHistory = ""
		s.CalcPrevNum = 0
		s.CalcOp = ""
		s.CalcNewNum = true
	case "⌫":
		if len(s.CalcDisplay) > 1 {
			s.CalcDisplay = s.CalcDisplay[:len(s.CalcDisplay)-1]
		} else {
			s.CalcDisplay = "0"
			s.CalcNewNum = true
		}
	case "+", "-", "×", "÷":
		val, _ := strconv.ParseFloat(s.CalcDisplay, 64)
		s.CalcPrevNum = val
		s.CalcOp = label
		s.CalcHistory = fmt.Sprintf("%.2g %s", val, label)
		s.CalcNewNum = true
	case "=":
		if s.CalcOp != "" {
			cur, _ := strconv.ParseFloat(s.CalcDisplay, 64)
			var res float64
			switch s.CalcOp {
			case "+":
				res = s.CalcPrevNum + cur
			case "-":
				res = s.CalcPrevNum - cur
			case "×":
				res = s.CalcPrevNum * cur
			case "÷":
				if cur != 0 {
					res = s.CalcPrevNum / cur
				}
			}
			s.CalcHistory = fmt.Sprintf("%.2g %s %.2g =", s.CalcPrevNum, s.CalcOp, cur)
			s.CalcDisplay = fmt.Sprintf("%.4g", res)
			s.CalcOp = ""
			s.CalcNewNum = true
		}
	case ".":
		if s.CalcNewNum {
			s.CalcDisplay = "0."
			s.CalcNewNum = false
		} else if !strings.Contains(s.CalcDisplay, ".") {
			s.CalcDisplay += "."
		}
	default:
		if s.CalcNewNum || s.CalcDisplay == "0" {
			s.CalcDisplay = label
			s.CalcNewNum = false
		} else {
			s.CalcDisplay += label
		}
	}
}

func (s *AppState) ToggleMusicPlayback() {
	s.IsPlayingMusic = !s.IsPlayingMusic
	if s.IsPlayingMusic {
		s.SetStatus("▶️ গান বাজছে")
		s.AddIPCLog("MusicService", "Playback started: "+s.Tracks[s.CurrentTrack].Title)
		PlayUnlockTone()
	} else {
		s.SetStatus("⏸️ গান থামানো হয়েছে")
		s.AddIPCLog("MusicService", "Playback paused")
		PlayLockTone()
	}
}

func (s *AppState) NextMusicTrack() {
	if len(s.Tracks) > 0 {
		s.CurrentTrack = (s.CurrentTrack + 1) % len(s.Tracks)
		s.MusicProgress = 0
		s.SetStatus("পরবর্তী গান: " + s.Tracks[s.CurrentTrack].Title)
		s.AddIPCLog("MusicService", "Track changed: "+s.Tracks[s.CurrentTrack].Title)
		PlayClickTone()
	}
}

func (s *AppState) PrevMusicTrack() {
	if len(s.Tracks) > 0 {
		s.CurrentTrack = (s.CurrentTrack - 1 + len(s.Tracks)) % len(s.Tracks)
		s.MusicProgress = 0
		s.SetStatus("পূর্ববর্তী গান: " + s.Tracks[s.CurrentTrack].Title)
		s.AddIPCLog("MusicService", "Track changed: "+s.Tracks[s.CurrentTrack].Title)
		PlayClickTone()
	}
}

func (s *AppState) ExecuteTerminalCommand() {
	cmd := strings.TrimSpace(s.TerminalInput)
	s.TerminalInput = ""
	s.TerminalLines = append(s.TerminalLines, "nil> "+cmd)

	switch strings.ToLower(cmd) {
	case "help":
		s.TerminalLines = append(s.TerminalLines,
			"Available Commands:",
			"  apps      - List registered NilOS applications",
			"  ps        - Show active processes and threads",
			"  mem       - Show RAM usage & garbage collector stats",
			"  hal       - Query NilHAL hardware state",
			"  clear     - Clear terminal buffer",
			"  version   - Show NilOS kernel and NilLang version",
		)
	case "apps":
		s.TerminalLines = append(s.TerminalLines,
			"Registered Alap Apps:",
			"  • org.onuron.notes      (v2.4.0) [Active]",
			"  • org.onuron.calculator (v1.2.0) [Standby]",
			"  • org.onuron.pos        (v3.1.0) [Standby]",
			"  • org.onuron.music      (v1.0.0) [Standby]",
			"  • org.onuron.terminal   (v2.0.0) [Active]",
		)
	case "ps":
		s.TerminalLines = append(s.TerminalLines,
			"PID   NAME              MEM(MB)  STATE",
			"1     kernel_main       1.2MB    RUNNING",
			"2     softbus_broker    0.8MB    SLEEPING",
			"10    alap_ui_engine    6.4MB    RUNNING",
			"12    notes_app         4.1MB    RUNNING",
		)
	case "mem":
		s.TerminalLines = append(s.TerminalLines,
			"Memory Heap Statistics:",
			"  Allocated: 14.2 MB",
			"  Total Sys: 28.5 MB",
			"  GC Cycles: 14 (pause < 0.2ms)",
		)
	case "hal":
		s.TerminalLines = append(s.TerminalLines,
			fmt.Sprintf("NilHAL State: Battery=%d%% Net=%s Latency=%dms", s.HAL.BatteryLevel, s.HAL.NetworkMode, s.HAL.NetworkLatency),
		)
	case "clear":
		s.TerminalLines = []string{
			"NilOS Interactive Shell REPL",
		}
	case "version":
		s.TerminalLines = append(s.TerminalLines,
			"NilOS v2.0 (x86_64-onuron) • NilLang Compiler v0.9.4 • Alap UI v2.0",
		)
	case "":
		// empty line
	default:
		s.TerminalLines = append(s.TerminalLines, fmt.Sprintf("=> %s (evaluated NilLang expr)", cmd))
	}

	if len(s.TerminalLines) > 30 {
		s.TerminalLines = s.TerminalLines[len(s.TerminalLines)-30:]
	}
	s.AddIPCLog("REPL", "Executed: "+cmd)
	PlayClickTone()
}

