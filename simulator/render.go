package main

import (
	"fmt"
	"math"
	"syscall"
	"time"
)

// Figma 3D Fluid Purple Glass Theme (matching Onuron OS nilsim design)
var (
	ColorCanvasBg      = RGB(11, 8, 26)   // Obsidian Violet Space (0x0B081A)
	ColorTitaniumEdge  = RGB(77, 58, 132)  // Luminous Violet Rim (0x4D3A84)
	ColorTitaniumInner = RGB(30, 21, 61)   // Frosted Purple Chassis (0x1E153D)
	ColorScreenBg      = RGB(20, 14, 40)   // Figma Deep Fluid Violet (0x140E28)
	ColorDockBg        = RGB(38, 26, 72)   // Frosted Violet Glass Dock (0x261A48)
	ColorSurface       = RGB(30, 21, 61)   // Frosted Purple Card Surface (0x1E153D)
	ColorSurfaceAlt    = RGB(49, 35, 95)   // Active Purple Glass Surface (0x31235F)
	ColorBorder        = RGB(77, 58, 132)  // Luminous Violet Border (0x4D3A84)
	ColorBorderGlow    = RGB(120, 90, 200) // Glowing Luminous Highlight
	ColorAccentIndigo  = RGB(99, 102, 241) // Vivid Indigo (0x6366F1)
	ColorCyan          = RGB(56, 189, 248) // Electric Sky Cyan (0x38BDF8)
	ColorEmerald       = RGB(34, 197, 94)  // Emerald Green (0x22C55E)
	ColorAmber         = RGB(251, 191, 36) // Warm Amber (0xFBBF24)
	ColorPurple        = RGB(192, 132, 252)// Figma Vibrant Violet (0xC084FC)
	ColorRose          = RGB(244, 63, 94)  // Rose Red (0xF43F5E)
	ColorTextHigh      = RGB(248, 250, 252)// Crisp Soft White (0xF8FAFC)
	ColorTextMed       = RGB(165, 180, 252)// Pastel Lavender (0xA5B4FC)
	ColorTextDim       = RGB(99, 102, 241) // Dim Indigo Accent (0x6366F1)
	ColorAccentBg      = RGB(49, 46, 129)  // Deep Indigo Glass Card (0x312E81)
	ColorTermBg        = RGB(11, 8, 26)   // Obsidian Violet Terminal (0x0B081A)
	ColorTermGreen     = RGB(74, 222, 128) // Emerald Shell Green
	ColorReceiptBg     = RGB(254, 252, 232)// Thermal Paper White
	ColorReceiptText   = RGB(20, 20, 20)
	ColorToolbarBg     = RGB(24, 16, 48)   // Floating Dark Violet Toolbar
	ColorToolbarBorder = RGB(77, 58, 132)
	ColorBlack         = RGB(0, 0, 0)
	ColorWhite         = RGB(255, 255, 255)
)

func GetCategoryColor(colorName string) uint32 {
	switch colorName {
	case "emerald":
		return ColorEmerald
	case "amber":
		return ColorAmber
	case "rose":
		return ColorRose
	case "violet":
		return ColorPurple
	case "cyan":
		return ColorCyan
	default:
		return ColorAccentIndigo
	}
}

func ToBengaliDigits(s string) string {
	var res []rune
	for _, ch := range s {
		switch ch {
		case '0':
			res = append(res, '০')
		case '1':
			res = append(res, '১')
		case '2':
			res = append(res, '২')
		case '3':
			res = append(res, '৩')
		case '4':
			res = append(res, '৪')
		case '5':
			res = append(res, '৫')
		case '6':
			res = append(res, '৬')
		case '7':
			res = append(res, '৭')
		case '8':
			res = append(res, '৮')
		case '9':
			res = append(res, '৯')
		default:
			res = append(res, ch)
		}
	}
	return string(res)
}

type Renderer struct {
	FontTitle   syscall.Handle
	FontBody    syscall.Handle
	FontHeader  syscall.Handle
	FontSmall   syscall.Handle
	FontClock   syscall.Handle
	FontHuge    syscall.Handle
	FontMono    syscall.Handle
	FontIcons   syscall.Handle
	FontTerm    syscall.Handle
}

func NewRenderer() *Renderer {
	return &Renderer{
		FontTitle:  CreateNativeFont("Segoe UI", 16, true),
		FontBody:   CreateNativeFont("Segoe UI", 13, false),
		FontHeader: CreateNativeFont("Segoe UI", 19, true),
		FontSmall:  CreateNativeFont("Segoe UI", 11, false),
		FontClock:  CreateNativeFont("Segoe UI", 13, true),
		FontHuge:   CreateNativeFont("Segoe UI", 42, true),
		FontMono:   CreateNativeFont("Consolas", 12, false),
		FontIcons:  CreateNativeFont("Segoe UI Symbol", 16, false),
		FontTerm:   CreateNativeFont("Consolas", 12, false),
	}
}

func (r *Renderer) Render(hdc syscall.Handle, width, height int32, state *AppState) {
	// 1. Draw Space Workspace Canvas with subtle gradient
	FillSolidRect(hdc, 0, 0, width, height, ColorCanvasBg)

	// Top Device Profile Switcher Bar
	r.drawTopProfileBar(hdc, 30, 10, width-60, state)

	profile := state.GetActiveProfile()
	phoneX := int32(30)
	phoneY := int32(48)
	phoneW := profile.Width
	phoneH := profile.Height

	screenX := phoneX + 12
	screenY := phoneY + 12
	screenW := phoneW - 24
	screenH := phoneH - 24

	// Thermal Printer Slot (If POS terminal mode)
	if profile.HasPrinter {
		r.drawThermalPrinterSlot(hdc, phoneX+20, phoneY-24, phoneW-40, state)
	}

	// 2. Physical Side Hardware Buttons (Luminous Titanium)
	DrawRoundedRect(hdc, phoneX-4, phoneY+100, 5, 30, 2, ColorBorder, ColorTitaniumInner, 1)
	DrawRoundedRect(hdc, phoneX-4, phoneY+145, 5, 48, 2, ColorBorder, ColorTitaniumInner, 1)
	DrawRoundedRect(hdc, phoneX-4, phoneY+205, 5, 48, 2, ColorBorder, ColorTitaniumInner, 1)
	DrawRoundedRect(hdc, phoneX+phoneW-1, phoneY+160, 5, 65, 2, ColorBorder, ColorTitaniumInner, 1)

	// 3. Phone Chassis Frame (Luminous Violet Border & 3D Glass)
	DrawRoundedRect(hdc, phoneX, phoneY, phoneW, phoneH, profile.BezelRadius, ColorTitaniumInner, ColorBorder, 2)

	// 4. OLED Display Screen
	if state.IsScreenOff {
		DrawRoundedRect(hdc, screenX, screenY, screenW, screenH, profile.BezelRadius-6, ColorBlack, ColorBlack, 1)
	} else {
		// Deep Fluid Violet Screen Background
		DrawRoundedRect(hdc, screenX, screenY, screenW, screenH, profile.BezelRadius-6, ColorScreenBg, ColorBlack, 1)

		// Status Bar & Dynamic Island
		r.drawStatusBar(hdc, screenX, screenY, screenW, state)
		r.drawDynamicIsland(hdc, screenX, screenY, screenW, state)

		if state.IsLocked {
			r.drawLockScreen(hdc, screenX, screenY, screenW, screenH, state)
		} else {
			switch state.ActiveAppID {
			case "home":
				r.drawHomeScreen(hdc, screenX, screenY, screenW, screenH, state)
			case "calc":
				r.drawCalculator(hdc, screenX, screenY, screenW, screenH, state)
			case "pos":
				r.drawPOSTerminal(hdc, screenX, screenY, screenW, screenH, state)
			case "music":
				r.drawMusicPlayer(hdc, screenX, screenY, screenW, screenH, state)
			case "terminal":
				r.drawTerminalREPL(hdc, screenX, screenY, screenW, screenH, state)
			case "devtools":
				r.drawDevTools(hdc, screenX, screenY, screenW, screenH, state)
			default:
				r.drawNotesApp(hdc, screenX, screenY, screenW, screenH, state)
			}
		}

		// Home Gesture Bar Indicator (Translucent rounded pill)
		homeBarW := int32(110)
		homeBarH := int32(4)
		homeBarX := screenX + (screenW-homeBarW)/2
		homeBarY := screenY + screenH - 8
		DrawRoundedRect(hdc, homeBarX, homeBarY, homeBarW, homeBarH, 2, ColorTextHigh, ColorTextHigh, 0)
	}

	// 5. Touch Ripple Indicator (Glowing cyan burst)
	if state.TouchAlpha > 0 {
		DrawRoundedRect(hdc, state.TouchX-12, state.TouchY-12, 24, 24, 12, ColorCyan, ColorWhite, 1)
		state.TouchAlpha--
	}

	// 6. Android Studio Floating Hardware Toolbar
	r.drawStudioToolbar(hdc, phoneX+phoneW+16, phoneY+50, state)
}

func (r *Renderer) drawTopProfileBar(hdc syscall.Handle, x, y, w int32, state *AppState) {
	tabX := x
	for i, p := range state.Profiles {
		tabW := int32(len([]rune(p.Name))*9 + 26)
		tabH := int32(28)

		if i == state.CurrentProfile {
			DrawGlassCard(hdc, tabX, y, tabW, tabH, 8, ColorPurple, ColorBorderGlow)
			DrawNativeText(hdc, p.Name, tabX, y, tabW, tabH, ColorBlack, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else {
			DrawGlassCard(hdc, tabX, y, tabW, tabH, 8, ColorSurface, ColorBorder)
			DrawNativeText(hdc, p.Name, tabX, y, tabW, tabH, ColorTextMed, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
		tabX += tabW + 8
	}

	// Active Architecture Badge on right
	archText := fmt.Sprintf("Onuron OS v2.0 • %d FPS • Figma 3D Glass", state.FPS)
	DrawNativeText(hdc, archText, x+w-280, y+4, 280, 24, ColorTextDim, r.FontSmall, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
}

func (r *Renderer) drawThermalPrinterSlot(hdc syscall.Handle, x, y, w int32, state *AppState) {
	DrawRoundedRect(hdc, x, y, w, 16, 4, ColorCanvasBg, ColorBorder, 1)
	DrawNativeText(hdc, "🖨️ Thermal ESC/POS Slot", x, y, w, 16, ColorTextDim, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	if len(state.HAL.PrintedReceipts) > 0 {
		rcptY := y - 40
		rcptW := w - 20
		DrawRoundedRect(hdc, x+10, rcptY, rcptW, 36, 4, ColorReceiptBg, ColorBorder, 1)
		DrawNativeText(hdc, "🏪 লখন ভাণ্ডার — ৳ ৫২০.০০ [OK]", x+14, rcptY+8, rcptW-8, 20, ColorReceiptText, r.FontSmall, DT_LEFT|DT_SINGLELINE)
	}
}

func (r *Renderer) drawStatusBar(hdc syscall.Handle, sx, sy, sw int32, state *AppState) {
	now := time.Now()
	clockStr := ToBengaliDigits(fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()))
	DrawNativeText(hdc, clockStr, sx+16, sy+12, 60, 20, ColorTextHigh, r.FontClock, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	netStr := fmt.Sprintf("%s 📶 %s%% 🔋", state.HAL.NetworkMode, ToBengaliDigits(fmt.Sprintf("%d", state.HAL.BatteryLevel)))
	DrawNativeText(hdc, netStr, sx+sw-130, sy+12, 114, 20, ColorTextMed, r.FontSmall, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
}

func (r *Renderer) drawDynamicIsland(hdc syscall.Handle, sx, sy, sw int32, state *AppState) {
	islandW := int32(124)
	islandH := int32(28)
	islandX := sx + (sw-islandW)/2
	islandY := sy + 8

	// Black obsidian pill with glass sheen
	DrawRoundedRect(hdc, islandX, islandY, islandW, islandH, 14, ColorBlack, ColorBorder, 1)
	sheen := BlendColors(ColorBlack, RGB(255, 255, 255), 40)
	DrawRoundedRect(hdc, islandX+10, islandY+2, islandW-20, 2, 1, sheen, sheen, 0)

	// Sensor and camera dots
	DrawRoundedRect(hdc, islandX+10, islandY+8, 12, 12, 6, RGB(25, 20, 45), ColorBlack, 1)

	statusTxt := "অনুরণ ওএস"
	if time.Now().Before(state.StatusMessageExp) && state.StatusMessage != "" {
		statusTxt = state.StatusMessage
	}
	DrawNativeText(hdc, statusTxt, islandX+26, islandY, islandW-30, islandH, ColorTextHigh, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
}

func (r *Renderer) drawLockScreen(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	now := time.Now()
	clockStr := ToBengaliDigits(fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()))
	dateStr := "বুধবার, ৮ অক্টোবর ২০২৬"

	DrawNativeText(hdc, "🔒", sx, sy+80, sw, 30, ColorTextMed, r.FontBody, DT_CENTER|DT_SINGLELINE)
	DrawNativeText(hdc, clockStr, sx, sy+115, sw, 55, ColorTextHigh, r.FontHuge, DT_CENTER|DT_SINGLELINE)
	DrawNativeText(hdc, dateStr, sx, sy+175, sw, 25, ColorTextMed, r.FontBody, DT_CENTER|DT_SINGLELINE)

	wX := sx + 20
	wY := sy + 230
	wW := sw - 40
	wH := int32(76)
	DrawGlassCard(hdc, wX, wY, wW, wH, 16, ColorSurface, ColorBorder)
	DrawNativeText(hdc, "📝 নীলাং নোটবুক (Alap Runtime)", wX+14, wY+14, wW-28, 20, ColorTextHigh, r.FontTitle, DT_LEFT|DT_SINGLELINE)
	DrawNativeText(hdc, fmt.Sprintf("%sটি সক্রিয় নোট • SoftBus Mesh Ready", ToBengaliDigits(fmt.Sprintf("%d", len(state.Notes)))), wX+14, wY+40, wW-28, 20, ColorTextMed, r.FontSmall, DT_LEFT|DT_SINGLELINE)

	DrawNativeText(hdc, "👆 আনলক করতে ক্লিক করুন", sx, sy+sh-80, sw, 30, ColorCyan, r.FontBody, DT_CENTER|DT_SINGLELINE)
}

// drawHomeScreen renders the 4-Column Squircle App Grid & Translucent Bottom Dock
func (r *Renderer) drawHomeScreen(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "অনুরণ অ্যাপস", sx+20, sy+44, sw-40, 22, ColorTextMed, r.FontTitle, DT_LEFT|DT_SINGLELINE)

	type AppIconItem struct {
		ID     string
		Symbol string
		Name   string
		Color  uint32
	}

	apps := []AppIconItem{
		{"notes", "📝", "নোটস", ColorCyan},
		{"calc", "🧮", "ক্যালক", ColorEmerald},
		{"pos", "🏪", "পিওএস", ColorAmber},
		{"music", "🎵", "মিউজিক", ColorPurple},
		{"terminal", "💻", "টার্মিনাল", ColorTermGreen},
		{"devtools", "🛠️", "লগ", ColorAccentIndigo},
		{"phone", "📞", "ফোন", ColorEmerald},
		{"messages", "💬", "বার্তা", ColorCyan},
		{"files", "📁", "ফাইলস", ColorAmber},
		{"settings", "⚙️", "সেটিংস", ColorPurple},
		{"softbus", "🌐", "সফটবাস", ColorAccentIndigo},
		{"nilpkg", "📦", "স্টোর", ColorRose},
	}

	gridStartY := sy + 76
	cols := 4
	iconSize := int32(48)
	gapX := (sw - 40 - int32(cols)*iconSize) / int32(cols-1)
	gapY := int32(24)

	for i, a := range apps {
		col := i % cols
		row := i / cols
		ix := sx + 20 + int32(col)*(iconSize+gapX)
		iy := gridStartY + int32(row)*(iconSize+gapY+16)

		if iy+iconSize > sy+sh-90 {
			break
		}

		// Draw Squircle Icon
		DrawSquircleIcon(hdc, ix, iy, iconSize, a.Color)
		DrawNativeText(hdc, a.Symbol, ix, iy+8, iconSize, 24, ColorTextHigh, r.FontHeader, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		DrawNativeText(hdc, a.Name, ix-8, iy+iconSize+4, iconSize+16, 16, ColorTextHigh, r.FontSmall, DT_CENTER|DT_SINGLELINE)
	}

	// Translucent Frosted Glass Bottom Dock
	dockW := sw - 32
	dockH := int32(64)
	dockX := sx + 16
	dockY := sy + sh - 82
	DrawGlassCard(hdc, dockX, dockY, dockW, dockH, 20, ColorDockBg, ColorBorder)

	dockApps := []AppIconItem{
		{"phone", "📞", "ফোন", ColorEmerald},
		{"messages", "💬", "বার্তা", ColorCyan},
		{"notes", "📝", "নোটস", ColorPurple},
		{"terminal", "💻", "শেল", ColorTermGreen},
	}

	dockGap := (dockW - int32(len(dockApps))*iconSize) / int32(len(dockApps)+1)
	for i, d := range dockApps {
		dix := dockX + dockGap + int32(i)*(iconSize+dockGap)
		diy := dockY + 8
		DrawSquircleIcon(hdc, dix, diy, iconSize, d.Color)
		DrawNativeText(hdc, d.Symbol, dix, diy+8, iconSize, 24, ColorTextHigh, r.FontHeader, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
}

func (r *Renderer) drawNotesApp(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	headerY := sy + 42
	DrawNativeText(hdc, "📝 নীলাং নোটবুক", sx+16, headerY, sw-32, 24, ColorTextHigh, r.FontHeader, DT_LEFT|DT_SINGLELINE)
	DrawNativeText(hdc, "Alap Framework • Reactive Store", sx+16, headerY+24, sw-32, 16, ColorTextMed, r.FontSmall, DT_LEFT|DT_SINGLELINE)

	// Search bar (Frosted glass)
	searchY := headerY + 44
	searchX := sx + 16
	searchW := sw - 32
	searchH := int32(34)
	DrawGlassCard(hdc, searchX, searchY, searchW, searchH, 10, ColorSurface, ColorBorder)
	searchTxt := "🔍 " + state.SearchQuery
	if state.ActiveInputField == "search" && state.CursorBlink {
		searchTxt += "|"
	}
	if state.SearchQuery == "" && state.ActiveInputField != "search" {
		searchTxt = "🔍 নোট বা ট্যাগ দিয়ে খুঁজুন..."
	}
	DrawNativeText(hdc, searchTxt, searchX+10, searchY, searchW-20, searchH, ColorTextMed, r.FontBody, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	// Category Chips
	chipsY := searchY + 40
	categories := []struct {
		id    string
		label string
	}{
		{"all", "📋 সব"},
		{"important", "⭐ জরুরি"},
		{"work", "💼 কাজ"},
		{"personal", "👤 ব্যক্তিগত"},
		{"ideas", "💡 আইডিয়া"},
		{"trash", "🗑️ ট্র্যাশ"},
	}

	chipX := sx + 16
	for _, cat := range categories {
		chipW := int32(len([]rune(cat.label))*10 + 16)
		chipH := int32(24)

		if cat.id == state.ActiveCategory {
			DrawGlassCard(hdc, chipX, chipsY, chipW, chipH, 12, ColorPurple, ColorBorderGlow)
			DrawNativeText(hdc, cat.label, chipX, chipsY, chipW, chipH, ColorBlack, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else {
			DrawGlassCard(hdc, chipX, chipsY, chipW, chipH, 12, ColorSurface, ColorBorder)
			DrawNativeText(hdc, cat.label, chipX, chipsY, chipW, chipH, ColorTextMed, r.FontSmall, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
		chipX += chipW + 6
		if chipX > sx+sw-20 {
			break
		}
	}

	// Notes Cards List (Glass cards with left accent border and specular sheen)
	cardsStartY := chipsY + 32
	notes := state.FilteredNotes()
	cardY := cardsStartY - int32(state.ScrollOffsetY)
	cardX := sx + 16
	cardW := sw - 32

	for _, note := range notes {
		cardH := int32(105)
		if len(note.Checklist) > 0 {
			cardH = 130
		}

		if cardY+cardH >= cardsStartY && cardY <= sy+sh-60 {
			catColor := GetCategoryColor(note.Color)
			DrawGlassCard(hdc, cardX, cardY, cardW, cardH, 14, ColorSurface, ColorBorder)
			// Accent colored top bar
			DrawRoundedRect(hdc, cardX+3, cardY+2, cardW-6, 3, 1, catColor, catColor, 0)

			titleText := note.Title
			if note.IsPinned {
				titleText = "📌 " + titleText
			}
			DrawNativeText(hdc, titleText, cardX+12, cardY+8, cardW-24, 22, ColorTextHigh, r.FontTitle, DT_LEFT|DT_SINGLELINE|DT_END_ELLIPSIS)
			DrawNativeText(hdc, note.Body, cardX+12, cardY+30, cardW-24, 32, ColorTextMed, r.FontBody, DT_LEFT|DT_WORDBREAK|DT_END_ELLIPSIS)

			if len(note.Checklist) > 0 {
				c := note.Checklist[0]
				checkMark := "[  ]"
				if c.Completed {
					checkMark = "[✓]"
				}
				checkStr := fmt.Sprintf("%s %s (%d/%d সম্পন্ন)", checkMark, c.Text, countCompleted(note.Checklist), len(note.Checklist))
				DrawNativeText(hdc, checkStr, cardX+12, cardY+66, cardW-24, 20, ColorEmerald, r.FontSmall, DT_LEFT|DT_SINGLELINE|DT_END_ELLIPSIS)
			}

			prioStr := "⚡ সাধারণ"
			if note.Priority == "high" {
				prioStr = "🔥 উচ্চ"
			} else if note.Priority == "low" {
				prioStr = "🌿 কম"
			}
			DrawNativeText(hdc, prioStr, cardX+12, cardY+cardH-24, 80, 18, ColorAmber, r.FontSmall, DT_LEFT|DT_SINGLELINE)
			DrawNativeText(hdc, note.UpdatedAt, cardX+cardW-100, cardY+cardH-24, 88, 18, ColorTextDim, r.FontSmall, DT_RIGHT|DT_SINGLELINE)
		}
		cardY += cardH + 10
	}

	// FAB (+) Button with vibrant glow
	fabX := sx + sw - 60
	fabY := sy + sh - 64
	DrawGlassCard(hdc, fabX, fabY, 46, 46, 23, ColorCyan, ColorWhite)
	DrawNativeText(hdc, "+", fabX, fabY-2, 46, 46, ColorBlack, r.FontHeader, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Note Editor Modal
	if state.IsEditing {
		r.drawNoteEditorModal(hdc, sx, sy, sw, sh, state)
	}
}

func (r *Renderer) drawNoteEditorModal(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawGlassCard(hdc, sx, sy+38, sw, sh-38, 20, ColorSurface, ColorBorder)

	DrawNativeText(hdc, "← ফিরে যান", sx+16, sy+48, 80, 24, ColorCyan, r.FontTitle, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	DrawNativeText(hdc, "নোট সম্পাদনা", sx+100, sy+48, sw-200, 24, ColorTextHigh, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	DrawNativeText(hdc, "সংরক্ষণ", sx+sw-76, sy+48, 60, 24, ColorEmerald, r.FontTitle, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)

	// Title Input
	DrawNativeText(hdc, "শিরোনাম:", sx+16, sy+84, sw-32, 18, ColorTextDim, r.FontSmall, DT_LEFT|DT_SINGLELINE)
	titleBoxY := sy + 104
	DrawGlassCard(hdc, sx+16, titleBoxY, sw-32, 34, 8, ColorSurfaceAlt, ColorBorder)
	titleVal := state.EditingNote.Title
	if state.ActiveInputField == "title" && state.CursorBlink {
		titleVal += "|"
	}
	DrawNativeText(hdc, titleVal, sx+24, titleBoxY, sw-48, 34, ColorTextHigh, r.FontTitle, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	// Body Input
	DrawNativeText(hdc, "বিস্তারিত নোট:", sx+16, sy+148, sw-32, 18, ColorTextDim, r.FontSmall, DT_LEFT|DT_SINGLELINE)
	bodyBoxY := sy + 168
	DrawGlassCard(hdc, sx+16, bodyBoxY, sw-32, 160, 8, ColorSurfaceAlt, ColorBorder)
	bodyVal := state.EditingNote.Body
	if state.ActiveInputField == "body" && state.CursorBlink {
		bodyVal += "|"
	}
	DrawNativeText(hdc, bodyVal, sx+24, bodyBoxY+8, sw-48, 144, ColorTextMed, r.FontBody, DT_LEFT|DT_WORDBREAK)

	// Checklist
	checkY := bodyBoxY + 172
	DrawNativeText(hdc, "✅ টাস্ক তালিকা:", sx+16, checkY, sw-32, 18, ColorTextDim, r.FontSmall, DT_LEFT|DT_SINGLELINE)
	for i, c := range state.EditingNote.Checklist {
		cBoxY := checkY + 22 + int32(i*26)
		checkMark := "[ ]"
		if c.Completed {
			checkMark = "[✓]"
		}
		cStr := fmt.Sprintf("%s %s", checkMark, c.Text)
		DrawNativeText(hdc, cStr, sx+24, cBoxY, sw-48, 22, ColorEmerald, r.FontBody, DT_LEFT|DT_SINGLELINE)
	}

	// Delete Button
	delY := sy + sh - 60
	DrawGlassCard(hdc, sx+16, delY, sw-32, 36, 8, ColorRose, ColorRose)
	DrawNativeText(hdc, "🗑️ নোট ডিলিট করুন", sx+16, delY, sw-32, 36, ColorTextHigh, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

func (r *Renderer) drawCalculator(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "🧮 নীলাং ক্যালকুলেটর", sx+20, sy+48, sw-40, 24, ColorTextHigh, r.FontHeader, DT_CENTER|DT_SINGLELINE)

	dispX := sx + 16
	dispY := sy + 84
	dispW := sw - 32
	dispH := int32(90)
	DrawGlassCard(hdc, dispX, dispY, dispW, dispH, 14, ColorSurface, ColorBorder)
	DrawNativeText(hdc, state.CalcHistory, dispX+12, dispY+10, dispW-24, 20, ColorTextDim, r.FontSmall, DT_RIGHT|DT_SINGLELINE)
	DrawNativeText(hdc, state.CalcDisplay, dispX+12, dispY+36, dispW-24, 44, ColorTextHigh, r.FontHuge, DT_RIGHT|DT_SINGLELINE)

	btnLabels := [][]string{
		{"AC", "⌫", "%", "÷"},
		{"7", "8", "9", "×"},
		{"4", "5", "6", "-"},
		{"1", "2", "3", "+"},
		{"0", ".", "=", "="},
	}

	gridY := dispY + dispH + 16
	btnSize := int32(52)
	gap := int32(10)

	for rowIdx, row := range btnLabels {
		for colIdx, lbl := range row {
			if rowIdx == 4 && colIdx == 3 {
				continue
			}
			bX := dispX + int32(colIdx)*(btnSize+gap)
			bY := gridY + int32(rowIdx)*(btnSize+gap)
			bW := btnSize
			if rowIdx == 4 && colIdx == 2 {
				bW = btnSize*2 + gap
			}

			bColor := ColorSurfaceAlt
			tColor := ColorTextHigh
			switch lbl {
			case "AC", "⌫", "%":
				bColor = RGB(60, 45, 100)
			case "÷", "×", "-", "+":
				bColor = ColorAccentIndigo
			case "=":
				bColor = ColorEmerald
				tColor = ColorBlack
			}

			DrawGlassCard(hdc, bX, bY, bW, btnSize, 14, bColor, ColorBorder)
			DrawNativeText(hdc, lbl, bX, bY, bW, btnSize, tColor, r.FontHeader, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
	}
}

func (r *Renderer) drawPOSTerminal(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "🏪 অনুরণ রিটেইল পিওএস", sx+16, sy+48, sw-32, 24, ColorTextHigh, r.FontHeader, DT_CENTER|DT_SINGLELINE)

	// POS Items
	items := []struct {
		name  string
		price float64
	}{
		{"চিনিকুড়া প্রিমিয়াম চাল ১ কেজি", 120.0},
		{"খাটি সরিষার তেল ৫০০ মিলি", 160.0},
		{"দার্জিলিং ব্ল্যাক টি ২৫০ গ্রাম", 240.0},
	}

	itY := sy + 84
	for _, itm := range items {
		DrawGlassCard(hdc, sx+16, itY, sw-32, 54, 10, ColorSurface, ColorBorder)
		DrawNativeText(hdc, itm.name, sx+26, itY+8, sw-120, 20, ColorTextHigh, r.FontTitle, DT_LEFT|DT_SINGLELINE)
		DrawNativeText(hdc, fmt.Sprintf("৳ %.2f", itm.price), sx+sw-110, itY+8, 80, 20, ColorEmerald, r.FontTitle, DT_RIGHT|DT_SINGLELINE)
		DrawNativeText(hdc, "বারকোড: 890103001 • স্টকে আছে", sx+26, itY+30, sw-52, 16, ColorTextDim, r.FontSmall, DT_LEFT|DT_SINGLELINE)
		itY += 62
	}

	// Checkout / Print Button
	btnY := sy + sh - 75
	DrawGlassCard(hdc, sx+16, btnY, sw-32, 48, 12, ColorEmerald, ColorBorderGlow)
	DrawNativeText(hdc, "🖨️ রিসিট প্রিন্ট করুন (৳ ৫২০.০০)", sx+16, btnY, sw-32, 48, ColorBlack, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

func (r *Renderer) drawMusicPlayer(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "🎵 অনুরণ মিউজিক প্লেয়ার", sx+16, sy+48, sw-32, 24, ColorTextHigh, r.FontHeader, DT_CENTER|DT_SINGLELINE)

	// Album Art Card (Figma 3D Purple Gradient)
	artY := sy + 84
	artW := sw - 60
	artH := int32(140)
	artX := sx + 30
	DrawGlassCard(hdc, artX, artY, artW, artH, 16, ColorAccentBg, ColorBorderGlow)
	DrawNativeText(hdc, "🎼 NilAudio Hi-Res", artX, artY+45, artW, 30, ColorTextHigh, r.FontHeader, DT_CENTER|DT_SINGLELINE)
	DrawNativeText(hdc, "96kHz • 24-bit Flac Lossless", artX, artY+75, artW, 20, ColorTextMed, r.FontSmall, DT_CENTER|DT_SINGLELINE)

	curTrack := state.Tracks[state.CurrentTrack]
	DrawNativeText(hdc, curTrack.Title, sx+16, artY+artH+14, sw-32, 24, ColorTextHigh, r.FontTitle, DT_CENTER|DT_SINGLELINE)
	DrawNativeText(hdc, curTrack.Artist+" • "+curTrack.Genre, sx+16, artY+artH+38, sw-32, 20, ColorCyan, r.FontSmall, DT_CENTER|DT_SINGLELINE)

	// Animated Audio Spectrum Wave Bars
	barY := artY + artH + 66
	numBars := 18
	barW := int32(6)
	gap := int32(6)
	startX := sx + (sw-int32(numBars)*(barW+gap))/2

	state.WaveformPhase = (state.WaveformPhase + 1) % 360
	for i := 0; i < numBars; i++ {
		bH := int32(6)
		if state.IsPlayingMusic {
			rad := float64(state.WaveformPhase+i*20) * math.Pi / 180.0
			bH = int32(10.0 + 16.0*math.Abs(math.Sin(rad)))
		}
		bx := startX + int32(i)*(barW+gap)
		DrawRoundedRect(hdc, bx, barY-bH/2, barW, bH, 3, ColorCyan, ColorCyan, 0)
	}

	// Playback Controls (Prev, Play/Pause, Next)
	ctrlY := barY + 30
	btnSize := int32(46)
	midX := sx + sw/2

	// Prev Button
	DrawGlassCard(hdc, midX-btnSize*2, ctrlY, btnSize, btnSize, 23, ColorSurfaceAlt, ColorBorder)
	DrawNativeText(hdc, "⏮️", midX-btnSize*2, ctrlY, btnSize, btnSize, ColorTextHigh, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Play/Pause Button
	playIcon := "▶️"
	if state.IsPlayingMusic {
		playIcon = "⏸️"
	}
	DrawGlassCard(hdc, midX-btnSize/2, ctrlY, btnSize, btnSize, 23, ColorCyan, ColorBorderGlow)
	DrawNativeText(hdc, playIcon, midX-btnSize/2, ctrlY, btnSize, btnSize, ColorBlack, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Next Button
	DrawGlassCard(hdc, midX+btnSize, ctrlY, btnSize, btnSize, 23, ColorSurfaceAlt, ColorBorder)
	DrawNativeText(hdc, "⏭️", midX+btnSize, ctrlY, btnSize, btnSize, ColorTextHigh, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Playlist list
	plistY := ctrlY + 60
	DrawNativeText(hdc, "প্লেলিস্ট:", sx+20, plistY, sw-40, 20, ColorTextDim, r.FontSmall, DT_LEFT|DT_SINGLELINE)
	for idx, t := range state.Tracks {
		pY := plistY + 22 + int32(idx*38)
		if pY > sy+sh-60 {
			break
		}
		bgCol := ColorSurface
		if idx == state.CurrentTrack {
			bgCol = ColorSurfaceAlt
		}
		DrawGlassCard(hdc, sx+16, pY, sw-32, 34, 8, bgCol, ColorBorder)
		prefix := "  "
		if idx == state.CurrentTrack {
			prefix = "▶ "
		}
		DrawNativeText(hdc, prefix+t.Title, sx+24, pY, sw-120, 34, ColorTextHigh, r.FontBody, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		DrawNativeText(hdc, t.Duration, sx+sw-70, pY, 50, 34, ColorTextDim, r.FontSmall, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
	}
}

func (r *Renderer) drawTerminalREPL(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "💻 NilLang & Linux REPL", sx+16, sy+48, sw-32, 24, ColorCyan, r.FontHeader, DT_LEFT|DT_SINGLELINE)

	termY := sy + 80
	termH := sh - 140
	DrawGlassCard(hdc, sx+16, termY, sw-32, termH, 10, ColorTermBg, ColorBorder)

	lY := termY + 12
	for _, line := range state.TerminalLines {
		if lY > termY+termH-40 {
			break
		}
		DrawNativeText(hdc, line, sx+24, lY, sw-48, 18, ColorTermGreen, r.FontTerm, DT_LEFT|DT_SINGLELINE)
		lY += 20
	}

	// Active Input Prompt
	promptY := termY + termH - 32
	promptTxt := "nil> " + state.TerminalInput
	if state.ActiveInputField == "terminal" && state.CursorBlink {
		promptTxt += "_"
	}
	DrawNativeText(hdc, promptTxt, sx+24, promptY, sw-48, 22, ColorTextHigh, r.FontTerm, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
}

func (r *Renderer) drawDevTools(hdc syscall.Handle, sx, sy, sw, sh int32, state *AppState) {
	DrawNativeText(hdc, "🛠️ NilOS DevTools & SoftBus Log", sx+16, sy+48, sw-32, 24, ColorCyan, r.FontTitle, DT_LEFT|DT_SINGLELINE)

	logBoxY := sy + 80
	logBoxH := sh - 140
	DrawGlassCard(hdc, sx+16, logBoxY, sw-32, logBoxH, 10, ColorTermBg, ColorBorder)

	lY := logBoxY + 10
	for i := len(state.IPCLogs) - 1; i >= 0 && lY < logBoxY+logBoxH-20; i-- {
		logText := state.IPCLogs[i]
		DrawNativeText(hdc, logText, sx+24, lY, sw-48, 18, ColorTextMed, r.FontMono, DT_LEFT|DT_SINGLELINE)
		lY += 20
	}
}

func (r *Renderer) drawStudioToolbar(hdc syscall.Handle, tbX, tbY int32, state *AppState) {
	tbW := int32(44)
	tbH := int32(520)
	DrawGlassCard(hdc, tbX, tbY, tbW, tbH, 20, ColorToolbarBg, ColorToolbarBorder)

	buttons := []struct {
		id    string
		icon  string
		color uint32
	}{
		{"power", "⏻", ColorRose},
		{"home_grid", "📱", ColorCyan},
		{"vol_up", "🔊", ColorTextHigh},
		{"vol_down", "🔉", ColorTextHigh},
		{"rotate", "🔄", ColorCyan},
		{"camera", "📸", ColorAmber},
		{"notes", "📝", ColorCyan},
		{"calc", "🧮", ColorEmerald},
		{"pos", "🏪", ColorAmber},
		{"music", "🎵", ColorPurple},
		{"terminal", "💻", ColorTermGreen},
		{"devtools", "🛠️", ColorPurple},
		{"back", "◁", ColorTextHigh},
		{"home", "○", ColorTextHigh},
		{"recents", "□", ColorTextHigh},
	}

	btnY := tbY + 10
	for _, b := range buttons {
		DrawNativeText(hdc, b.icon, tbX, btnY, tbW, 30, b.color, r.FontTitle, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		btnY += 34
	}
}

func countCompleted(items []ChecklistItem) int {
	c := 0
	for _, item := range items {
		if item.Completed {
			c++
		}
	}
	return c
}

