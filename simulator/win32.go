package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	pGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	pRegisterClassExW   = user32.NewProc("RegisterClassExW")
	pCreateWindowExW     = user32.NewProc("CreateWindowExW")
	pDefWindowProcW      = user32.NewProc("DefWindowProcW")
	pShowWindow          = user32.NewProc("ShowWindow")
	pUpdateWindow        = user32.NewProc("UpdateWindow")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pBeginPaint          = user32.NewProc("BeginPaint")
	pEndPaint            = user32.NewProc("EndPaint")
	pInvalidateRect      = user32.NewProc("InvalidateRect")
	pGetClientRect       = user32.NewProc("GetClientRect")
	pSetTimer            = user32.NewProc("SetTimer")
	pKillTimer           = user32.NewProc("KillTimer")
	pDrawTextW           = user32.NewProc("DrawTextW")
	pFillRect            = user32.NewProc("FillRect")
	pLoadCursorW         = user32.NewProc("LoadCursorW")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pRectangle              = gdi32.NewProc("Rectangle")
	pEllipse                = gdi32.NewProc("Ellipse")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
)

const (
	WM_DESTROY      = 0x0002
	WM_PAINT        = 0x000F
	WM_TIMER        = 0x0113
	WM_KEYDOWN      = 0x0100
	WM_CHAR         = 0x0102
	WM_LBUTTONDOWN  = 0x0201
	WM_LBUTTONUP    = 0x0202
	WM_MOUSEMOVE    = 0x0200
	WM_MOUSEWHEEL   = 0x020A
	WM_ERASEBKGND   = 0x0014

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CS_HREDRAW          = 0x0002
	CS_VREDRAW          = 0x0001
	CW_USEDEFAULT       = 0x80000000
	COLOR_WINDOW        = 5
	IDC_ARROW           = 32512

	SRCCOPY     = 0x00CC0020
	TRANSPARENT = 1
	OPAQUE      = 2

	DT_TOP              = 0x00000000
	DT_LEFT             = 0x00000000
	DT_CENTER           = 0x00000001
	DT_RIGHT            = 0x00000002
	DT_VCENTER          = 0x00000004
	DT_WORDBREAK        = 0x00000010
	DT_SINGLELINE       = 0x00000020
	DT_NOPREFIX         = 0x00000800
	DT_END_ELLIPSIS     = 0x00008000

	PS_SOLID = 0
	PS_NULL  = 5

	CLEARTYPE_QUALITY = 5
	ANTIALIASED_QUALITY = 4
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	HWnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

func RGB(r, g, b byte) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

func BlendColors(bg, fg uint32, alpha uint8) uint32 {
	if alpha == 0 {
		return bg
	}
	if alpha == 255 {
		return fg
	}
	a := uint32(alpha)
	invA := 255 - a

	rBg := bg & 0xFF
	gBg := (bg >> 8) & 0xFF
	bBg := (bg >> 16) & 0xFF

	rFg := fg & 0xFF
	gFg := (fg >> 8) & 0xFF
	bFg := (fg >> 16) & 0xFF

	r := (rFg*a + rBg*invA) / 255
	g := (gFg*a + gBg*invA) / 255
	b := (bFg*a + bBg*invA) / 255

	return r | (g << 8) | (b << 16)
}

func UTF16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return p
}

func FillSolidRect(hdc syscall.Handle, x, y, w, h int32, color uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(color))
	rc := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	pFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), brush)
	pDeleteObject.Call(brush)
}

func DrawRoundedRect(hdc syscall.Handle, x, y, w, h, radius int32, fillColor, borderColor uint32, borderWidth int32) {
	var pen uintptr
	if borderWidth > 0 {
		pen, _, _ = pCreatePen.Call(uintptr(PS_SOLID), uintptr(borderWidth), uintptr(borderColor))
	} else {
		pen, _, _ = pCreatePen.Call(uintptr(PS_NULL), 0, 0)
	}

	brush, _, _ := pCreateSolidBrush.Call(uintptr(fillColor))

	oldPen, _, _ := pSelectObject.Call(uintptr(hdc), pen)
	oldBrush, _, _ := pSelectObject.Call(uintptr(hdc), brush)

	pRoundRect.Call(uintptr(hdc), uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h), uintptr(radius*2), uintptr(radius*2))

	pSelectObject.Call(uintptr(hdc), oldPen)
	pSelectObject.Call(uintptr(hdc), oldBrush)
	pDeleteObject.Call(pen)
	pDeleteObject.Call(brush)
}

// DrawGlassCard renders a frosted glass card with 3D top specular sheen and luminous border
func DrawGlassCard(hdc syscall.Handle, x, y, w, h, radius int32, bg, border uint32) {
	// Base card
	DrawRoundedRect(hdc, x, y, w, h, radius, bg, border, 1)

	// Top edge glass sheen (thin highlight)
	sheenColor := BlendColors(bg, RGB(255, 255, 255), 45)
	if radius > 4 && w > radius*2 {
		DrawRoundedRect(hdc, x+radius/2, y+1, w-radius, 2, 1, sheenColor, sheenColor, 0)
	}
}

// DrawSquircleIcon renders a modern iOS/Figma squircle app icon with outer glow and glass sheen
func DrawSquircleIcon(hdc syscall.Handle, x, y, size int32, iconBg uint32) {
	r := int32(13)

	// Outer soft glow
	glow := BlendColors(RGB(20, 14, 40), iconBg, 50)
	DrawRoundedRect(hdc, x-2, y-2, size+4, size+4, r+2, glow, glow, 0)

	// Body
	DrawRoundedRect(hdc, x, y, size, size, r, iconBg, BlendColors(iconBg, RGB(255, 255, 255), 35), 1)

	// Glass top sheen
	sheen := BlendColors(iconBg, RGB(255, 255, 255), 55)
	DrawRoundedRect(hdc, x+3, y+2, size-6, size/3, r-3, sheen, sheen, 0)
}

func DrawNativeText(hdc syscall.Handle, text string, x, y, w, h int32, color uint32, font syscall.Handle, flags uint32) {
	if text == "" {
		return
	}
	pSetBkMode.Call(uintptr(hdc), uintptr(TRANSPARENT))
	pSetTextColor.Call(uintptr(hdc), uintptr(color))

	var oldFont uintptr
	if font != 0 {
		oldFont, _, _ = pSelectObject.Call(uintptr(hdc), uintptr(font))
	}

	rc := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	pText := UTF16Ptr(text)
	pDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(pText)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), uintptr(flags))

	if font != 0 {
		pSelectObject.Call(uintptr(hdc), oldFont)
	}
}

func CreateNativeFont(name string, size int32, bold bool) syscall.Handle {
	weight := 400
	if bold {
		weight = 700
	}
	fontName := UTF16Ptr(name)
	hFont, _, _ := pCreateFontW.Call(
		uintptr(size),
		0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		1, // DEFAULT_CHARSET
		0, 0,
		uintptr(CLEARTYPE_QUALITY), // ClearType anti-aliasing
		0,
		uintptr(unsafe.Pointer(fontName)),
	)
	return syscall.Handle(hFont)
}

