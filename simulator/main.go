package main

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	appState *AppState
	renderer *Renderer
	mainHWnd syscall.Handle
)

func init() {
	runtime.LockOSThread()
}

func main() {
	appState = NewAppState()
	renderer = NewRenderer()

	hInstance, _, _ := pGetModuleHandleW.Call(0)
	className := UTF16Ptr("NilPhoneNativeWindow")
	windowTitle := UTF16Ptr("NilPhone 16 Pro Max — Pure Native Go Emulator (Win32 GDI)")

	cursor, _, _ := pLoadCursorW.Call(0, uintptr(IDC_ARROW))

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(WndProc),
		HInstance:     syscall.Handle(hInstance),
		HCursor:       syscall.Handle(cursor),
		HbrBackground: 0,
		LpszClassName: className,
	}

	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	winW := int32(560)
	winH := int32(920)

	hwnd, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		CW_USEDEFAULT, CW_USEDEFAULT,
		uintptr(winW), uintptr(winH),
		0, 0, hInstance, 0,
	)

	mainHWnd = syscall.Handle(hwnd)

	pShowWindow.Call(uintptr(mainHWnd), 1)
	pUpdateWindow.Call(uintptr(mainHWnd))

	// Timer for cursor blink & clock (every 500ms)
	pSetTimer.Call(uintptr(mainHWnd), 1, 500, 0)

	fmt.Println("==============================================================")
	fmt.Println("  📱 NILPHONE 16 PRO MAX — 100% PURE NATIVE GO DESKTOP EMULATOR")
	fmt.Println("  ⚡ Architecture: Pure Win32 GDI+ (Zero HTML / Zero Browser)")
	fmt.Println("  🎯 Window: Standalone Native OS Window Active")
	fmt.Println("==============================================================")

	var msg MSG
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 || int32(ret) == -1 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func WndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		var rect RECT
		pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
		w := rect.Right - rect.Left
		h := rect.Bottom - rect.Top

		// FPS counter
		appState.FrameCount++
		if time.Since(appState.LastFrameTime) >= time.Second {
			appState.FPS = appState.FrameCount
			appState.FrameCount = 0
			appState.LastFrameTime = time.Now()
		}

		// Double-buffered rendering
		memDC, _, _ := pCreateCompatibleDC.Call(hdc)
		memBmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
		oldBmp, _, _ := pSelectObject.Call(memDC, memBmp)

		renderer.Render(syscall.Handle(memDC), w, h, appState)

		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memDC, 0, 0, SRCCOPY)

		pSelectObject.Call(memDC, oldBmp)
		pDeleteObject.Call(memBmp)
		pDeleteDC.Call(memDC)

		pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_ERASEBKGND:
		return 1

	case WM_TIMER:
		appState.CursorBlink = !appState.CursorBlink
		pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_LBUTTONDOWN:
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)
		appState.TouchX = x
		appState.TouchY = y
		appState.TouchAlpha = 3
		handleMouseClick(x, y)
		pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_CHAR:
		ch := rune(wParam)
		handleCharInput(ch)
		pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_KEYDOWN:
		handleKeyDown(wParam)
		pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_MOUSEWHEEL:
		delta := int16((wParam >> 16) & 0xFFFF)
		if delta > 0 {
			if appState.ScrollOffsetY > 0 {
				appState.ScrollOffsetY -= 30
			}
		} else {
			appState.ScrollOffsetY += 30
		}
		pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_DESTROY:
		pKillTimer.Call(uintptr(hwnd), 1)
		pPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func handleMouseClick(x, y int32) {
	// 1. Top Profile Switcher Tabs Click
	if y >= 10 && y <= 38 {
		tabX := int32(30)
		for i, p := range appState.Profiles {
			tabW := int32(len([]rune(p.Name))*10 + 24)
			if x >= tabX && x <= tabX+tabW {
				appState.SwitchProfile(i)
				return
			}
			tabX += tabW + 8
		}
	}

	profile := appState.GetActiveProfile()
	phoneX := int32(30)
	phoneY := int32(48)
	phoneW := profile.Width
	phoneH := profile.Height

	screenX := phoneX + 12
	screenY := phoneY + 12
	screenW := phoneW - 24
	screenH := phoneH - 24

	// 2. Android Studio Right Toolbar Click Detection
	tbX := phoneX + phoneW + 16
	tbY := phoneY + 50
	if x >= tbX && x <= tbX+44 && y >= tbY && y <= tbY+520 {
		btnIdx := (y - tbY - 10) / 34
		switch btnIdx {
		case 0: // Power
			if appState.IsScreenOff {
				PlayUnlockTone()
				appState.IsScreenOff = false
				appState.IsLocked = true
			} else {
				PlayLockTone()
				appState.IsScreenOff = true
				appState.IsLocked = true
			}
		case 1: // Home Grid (Squircle App Launcher)
			PlayClickTone()
			appState.ActiveAppID = "home"
			appState.IsEditing = false
			appState.IsLocked = false
			appState.SetStatus("অনুরণ অ্যাপস লঞ্চার")
		case 2: // Vol+
			if appState.HAL.Volume < 100 {
				appState.HAL.Volume += 5
				PlayClickTone()
				appState.SetStatus(fmt.Sprintf("Volume: %d%%", appState.HAL.Volume))
			}
		case 3: // Vol-
			if appState.HAL.Volume > 0 {
				appState.HAL.Volume -= 5
				PlayClickTone()
				appState.SetStatus(fmt.Sprintf("Volume: %d%%", appState.HAL.Volume))
			}
		case 4: // Rotate
			PlayClickTone()
			appState.IsLandscape = !appState.IsLandscape
			appState.SetStatus("Orientation Changed")
		case 5: // Screenshot
			PlayShutterTone()
			appState.SetStatus("📸 Screenshot Saved")
			appState.AddIPCLog("NilHAL.Camera", "Screen framebuffer captured to PNG")
		case 6: // Notes App
			PlayClickTone()
			appState.ActiveAppID = "notes"
			appState.IsLocked = false
		case 7: // Calc App
			PlayClickTone()
			appState.ActiveAppID = "calc"
			appState.IsLocked = false
		case 8: // POS App
			PlayClickTone()
			appState.ActiveAppID = "pos"
			appState.IsLocked = false
		case 9: // Music App
			PlayClickTone()
			appState.ActiveAppID = "music"
			appState.IsLocked = false
		case 10: // Terminal / REPL
			PlayClickTone()
			appState.ActiveAppID = "terminal"
			appState.ActiveInputField = "terminal"
			appState.IsLocked = false
		case 11: // DevTools
			PlayClickTone()
			appState.ActiveAppID = "devtools"
			appState.IsLocked = false
		case 12: // Back
			PlayClickTone()
			if appState.IsEditing {
				appState.IsEditing = false
			} else if appState.ActiveAppID != "home" && appState.ActiveAppID != "notes" {
				appState.ActiveAppID = "home"
			}
		case 13: // Home
			PlayClickTone()
			appState.IsEditing = false
			appState.ActiveAppID = "home"
			appState.SearchQuery = ""
			appState.ActiveCategory = "all"
			appState.SetStatus("NilOS Home")
		case 14: // Recents
			PlayClickTone()
			if appState.ActiveAppID == "home" {
				appState.ActiveAppID = "notes"
			} else if appState.ActiveAppID == "notes" {
				appState.ActiveAppID = "calc"
			} else if appState.ActiveAppID == "calc" {
				appState.ActiveAppID = "music"
			} else {
				appState.ActiveAppID = "home"
			}
		}
		return
	}

	// 3. Lock Screen Unlock Detection
	if appState.IsLocked {
		PlayUnlockTone()
		appState.IsLocked = false
		appState.SetStatus("Device Unlocked")
		return
	}

	// 4. Inside Display Screen Click Detection
	if x >= screenX && x <= screenX+screenW && y >= screenY && y <= screenY+screenH {
		// Home gesture bar click at bottom -> go to home
		if y >= screenY+screenH-20 {
			PlayClickTone()
			appState.ActiveAppID = "home"
			appState.IsEditing = false
			appState.SetStatus("অনুরণ অ্যাপস লঞ্চার")
			return
		}

		// Dynamic Island click
		if x >= screenX+80 && x <= screenX+screenW-80 && y >= screenY+8 && y <= screenY+36 {
			PlayClickTone()
			appState.SetStatus("Onuron OS 2.0 Native")
			return
		}

		// Home Screen Squircle App Grid & Dock Clicks
		if appState.ActiveAppID == "home" {
			gridStartY := screenY + 76
			cols := 4
			iconSize := int32(48)
			gapX := (screenW - 40 - int32(cols)*iconSize) / int32(cols-1)
			gapY := int32(24)

			appIDs := []string{
				"notes", "calc", "pos", "music",
				"terminal", "devtools", "notes", "notes",
				"notes", "devtools", "devtools", "devtools",
			}

			// App grid clicks
			for i, appId := range appIDs {
				col := i % cols
				row := i / cols
				ix := screenX + 20 + int32(col)*(iconSize+gapX)
				iy := gridStartY + int32(row)*(iconSize+gapY+16)

				if x >= ix && x <= ix+iconSize && y >= iy && y <= iy+iconSize+16 {
					PlayClickTone()
					appState.ActiveAppID = appId
					if appId == "terminal" {
						appState.ActiveInputField = "terminal"
					}
					return
				}
			}

			// Bottom Dock clicks
			dockW := screenW - 32
			dockH := int32(64)
			dockX := screenX + 16
			dockY := screenY + screenH - 82

			dockApps := []string{"notes", "notes", "notes", "terminal"}
			dockGap := (dockW - int32(len(dockApps))*iconSize) / int32(len(dockApps)+1)
			for i, dId := range dockApps {
				dix := dockX + dockGap + int32(i)*(iconSize+dockGap)
				diy := dockY + 8
				if x >= dix && x <= dix+iconSize && y >= diy && y <= diy+dockH {
					PlayClickTone()
					appState.ActiveAppID = dId
					if dId == "terminal" {
						appState.ActiveInputField = "terminal"
					}
					return
				}
			}
			return
		}

		// DevTools View Clicks
		if appState.ActiveAppID == "devtools" {
			return
		}

		// Terminal REPL View Clicks
		if appState.ActiveAppID == "terminal" {
			appState.ActiveInputField = "terminal"
			return
		}

		// Music Player Clicks
		if appState.ActiveAppID == "music" {
			artY := screenY + 88
			artH := int32(140)
			barY := artY + artH + 66
			ctrlY := barY + 30
			btnSize := int32(46)
			midX := screenX + screenW/2

			// Prev Button
			if x >= midX-btnSize*2 && x <= midX-btnSize && y >= ctrlY && y <= ctrlY+btnSize {
				appState.PrevMusicTrack()
				return
			}
			// Play/Pause Button
			if x >= midX-btnSize/2 && x <= midX+btnSize/2 && y >= ctrlY && y <= ctrlY+btnSize {
				appState.ToggleMusicPlayback()
				return
			}
			// Next Button
			if x >= midX+btnSize && x <= midX+btnSize*2 && y >= ctrlY && y <= ctrlY+btnSize {
				appState.NextMusicTrack()
				return
			}

			// Playlist items
			plistY := ctrlY + 60
			for idx := range appState.Tracks {
				pY := plistY + 22 + int32(idx*38)
				if y >= pY && y <= pY+34 {
					appState.CurrentTrack = idx
					appState.IsPlayingMusic = true
					appState.SetStatus("বাজছে: " + appState.Tracks[idx].Title)
					PlayUnlockTone()
					return
				}
			}
			return
		}

		// POS Terminal View Clicks
		if appState.ActiveAppID == "pos" {
			btnY := screenY + screenH - 75
			if y >= btnY && y <= btnY+48 {
				items := []string{
					"চিনিকুড়া প্রিমিয়াম চাল ১ কেজি - ৳১২০",
					"খাটি সরিষার তেল ৫০০ মিলি - ৳১৬০",
					"দার্জিলিং ব্ল্যাক টি ২৫০ গ্রাম - ৳২৪০",
				}
				appState.SimulateThermalReceipt("লখন ভাণ্ডার এন্টারপ্রাইজ", items, 520.0)
				return
			}
			return
		}

		// Note Editor Modal Clicks
		if appState.IsEditing {
			// Back button
			if x >= screenX+16 && x <= screenX+96 && y >= screenY+48 && y <= screenY+72 {
				PlayClickTone()
				appState.IsEditing = false
				return
			}
			// Save button
			if x >= screenX+screenW-80 && x <= screenX+screenW-16 && y >= screenY+48 && y <= screenY+72 {
				saveEditingNote()
				return
			}
			// Title input click
			if y >= screenY+104 && y <= screenY+138 {
				PlayClickTone()
				appState.ActiveInputField = "title"
				return
			}
			// Body input click
			if y >= screenY+168 && y <= screenY+328 {
				PlayClickTone()
				appState.ActiveInputField = "body"
				return
			}
			// Delete button
			delY := screenY + screenH - 60
			if y >= delY && y <= delY+36 {
				deleteActiveNote()
				return
			}
			return
		}

		// Calculator App Clicks
		if appState.ActiveAppID == "calc" {
			dispX := screenX + 16
			dispY := screenY + 88
			gridY := dispY + 90 + 16
			btnSize := int32(52)
			gap := int32(10)

			btnLabels := [][]string{
				{"AC", "⌫", "%", "÷"},
				{"7", "8", "9", "×"},
				{"4", "5", "6", "-"},
				{"1", "2", "3", "+"},
				{"0", ".", "=", "="},
			}

			for rowIdx, row := range btnLabels {
				for colIdx, lbl := range row {
					bX := dispX + int32(colIdx)*(btnSize+gap)
					bY := gridY + int32(rowIdx)*(btnSize+gap)
					bW := btnSize
					if rowIdx == 4 && colIdx == 2 {
						bW = btnSize*2 + gap
					}

					if x >= bX && x <= bX+bW && y >= bY && y <= bY+btnSize {
						appState.HandleCalcButton(lbl)
						return
					}
				}
			}
			return
		}

		// Notes App Clicks
		// Search bar
		searchY := screenY + 42 + 46
		if y >= searchY && y <= searchY+34 {
			PlayClickTone()
			appState.ActiveInputField = "search"
			return
		}

		// Category Chips
		chipsY := searchY + 42
		if y >= chipsY && y <= chipsY+26 {
			categories := []string{"all", "important", "work", "personal", "ideas", "trash"}
			labels := []string{"📋 সব", "⭐ জরুরি", "💼 কাজ", "👤 ব্যক্তিগত", "💡 আইডিয়া", "🗑️ ট্র্যাশ"}
			chipX := screenX + 16
			for i, cat := range categories {
				chipW := int32(len([]rune(labels[i]))*11 + 16)
				if x >= chipX && x <= chipX+chipW {
					PlayClickTone()
					appState.ActiveCategory = cat
					return
				}
				chipX += chipW + 6
			}
			return
		}

		// FAB (+) Button
		fabX := screenX + screenW - 60
		fabY := screenY + screenH - 64
		if x >= fabX && x <= fabX+46 && y >= fabY && y <= fabY+46 {
			openNewNoteEditor()
			return
		}

		// Note Cards List Click & Checkbox Toggle
		cardsStartY := chipsY + 32
		notes := appState.FilteredNotes()
		cardY := cardsStartY - int32(appState.ScrollOffsetY)
		cardX := screenX + 16
		cardW := screenW - 32

		for _, note := range notes {
			cardH := int32(105)
			if len(note.Checklist) > 0 {
				cardH = 130
			}

			if x >= cardX && x <= cardX+cardW && y >= cardY && y <= cardY+cardH {
				if len(note.Checklist) > 0 && y >= cardY+65 && y <= cardY+90 {
					toggleNoteChecklist(note.ID, 0)
					return
				}
				openExistingNoteEditor(note)
				return
			}
			cardY += cardH + 10
		}
	}
}

func handleCharInput(ch rune) {
	if ch == 8 {
		return
	}
	if ch < 32 && ch != 10 && ch != 13 {
		return
	}
	PlayClickTone()

	if appState.ActiveAppID == "terminal" {
		appState.TerminalInput += string(ch)
		return
	}

	if appState.IsEditing {
		if appState.ActiveInputField == "title" {
			appState.EditingNote.Title += string(ch)
		} else if appState.ActiveInputField == "body" {
			appState.EditingNote.Body += string(ch)
		}
	} else if appState.ActiveInputField == "search" {
		appState.SearchQuery += string(ch)
	}
}

func handleKeyDown(wParam uintptr) {
	switch wParam {
	case 8: // Backspace
		PlayClickTone()
		if appState.ActiveAppID == "terminal" {
			if len(appState.TerminalInput) > 0 {
				r := []rune(appState.TerminalInput)
				appState.TerminalInput = string(r[:len(r)-1])
			}
			return
		}

		if appState.IsEditing {
			if appState.ActiveInputField == "title" && len(appState.EditingNote.Title) > 0 {
				r := []rune(appState.EditingNote.Title)
				appState.EditingNote.Title = string(r[:len(r)-1])
			} else if appState.ActiveInputField == "body" && len(appState.EditingNote.Body) > 0 {
				r := []rune(appState.EditingNote.Body)
				appState.EditingNote.Body = string(r[:len(r)-1])
			}
		} else if appState.ActiveInputField == "search" && len(appState.SearchQuery) > 0 {
			r := []rune(appState.SearchQuery)
			appState.SearchQuery = string(r[:len(r)-1])
		}
	case 13: // Enter
		if appState.ActiveAppID == "terminal" {
			appState.ExecuteTerminalCommand()
			return
		}
		PlayClickTone()
		if appState.IsEditing && appState.ActiveInputField == "body" {
			appState.EditingNote.Body += "\n"
		}
	case 27: // Escape
		PlayClickTone()
		if appState.IsEditing {
			appState.IsEditing = false
		}
	}
}

func openNewNoteEditor() {
	PlayClickTone()
	appState.EditingNote = Note{
		ID:        fmt.Sprintf("note-%d", time.Now().UnixNano()),
		Title:     "",
		Body:      "",
		Category:  "important",
		Priority:  "medium",
		Color:     "indigo",
		IsPinned:  false,
		Checklist: []ChecklistItem{},
		UpdatedAt: "এখনই",
	}
	appState.ActiveInputField = "title"
	appState.IsEditing = true
}

func openExistingNoteEditor(note Note) {
	PlayClickTone()
	appState.EditingNote = note
	appState.ActiveInputField = "body"
	appState.IsEditing = true
}

func saveEditingNote() {
	PlayClickTone()
	if strings.TrimSpace(appState.EditingNote.Title) == "" {
		appState.EditingNote.Title = "শিরোনামহীন নোট"
	}
	appState.EditingNote.UpdatedAt = "আজ " + time.Now().Format("15:04")

	found := false
	for i, n := range appState.Notes {
		if n.ID == appState.EditingNote.ID {
			appState.Notes[i] = appState.EditingNote
			found = true
			break
		}
	}
	if !found {
		appState.Notes = append([]Note{appState.EditingNote}, appState.Notes...)
	}

	appState.IsEditing = false
	appState.SetStatus("নোট সংরক্ষিত হয়েছে")
	appState.AddIPCLog("NotesEngine", "Note saved: "+appState.EditingNote.ID)
}

func deleteActiveNote() {
	PlayClickTone()
	var remaining []Note
	for _, n := range appState.Notes {
		if n.ID != appState.EditingNote.ID {
			remaining = append(remaining, n)
		}
	}
	appState.Notes = remaining
	appState.IsEditing = false
	appState.SetStatus("নোট মুছে ফেলা হয়েছে")
	appState.AddIPCLog("NotesEngine", "Note deleted")
}

func toggleNoteChecklist(noteID string, checkIdx int) {
	PlayClickTone()
	for i, n := range appState.Notes {
		if n.ID == noteID {
			if checkIdx < len(n.Checklist) {
				appState.Notes[i].Checklist[checkIdx].Completed = !appState.Notes[i].Checklist[checkIdx].Completed
				appState.SetStatus("টাস্ক আপডেট সম্পন্ন")
				appState.AddIPCLog("NotesEngine", "Checklist toggled")
			}
			break
		}
	}
}

