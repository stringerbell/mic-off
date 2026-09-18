//go:build windows

package recorder

// The native "Change hotkey" window, built directly on user32 so the
// Windows binaries stay cgo-free. All logic lives in Go (session.go); this
// file only draws the window, forwards key events and shows text.

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"micoff/internal/hotkey"
	"micoff/internal/keys"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW    = user32.NewProc("RegisterClassExW")
	pCreateWindowExW     = user32.NewProc("CreateWindowExW")
	pDefWindowProcW      = user32.NewProc("DefWindowProcW")
	pDestroyWindow       = user32.NewProc("DestroyWindow")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pSendMessageW        = user32.NewProc("SendMessageW")
	pSetWindowTextW      = user32.NewProc("SetWindowTextW")
	pEnableWindow        = user32.NewProc("EnableWindow")
	pGetKeyState         = user32.NewProc("GetKeyState")
	pLoadCursorW         = user32.NewProc("LoadCursorW")
	pGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	pAdjustWindowRectEx  = user32.NewProc("AdjustWindowRectEx")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pShowWindow          = user32.NewProc("ShowWindow")
	pCreateFontW         = gdi32.NewProc("CreateFontW")
	pDeleteObject        = gdi32.NewProc("DeleteObject")
	pGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
)

const (
	wsCaption = 0x00C00000
	wsSysMenu = 0x00080000
	wsVisible = 0x10000000
	wsChild   = 0x40000000
	wsTabStop = 0x00010000

	wsExDlgModalFrame = 0x1
	wsExTopmost       = 0x8

	ssCenter        = 0x1
	bsPushButton    = 0x0
	bsDefPushButton = 0x1

	wmDestroy    = 0x0002
	wmClose      = 0x0010
	wmSetFont    = 0x0030
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105
	wmCommand    = 0x0111

	swShow      = 5
	smCXScreen  = 0
	smCYScreen  = 1
	idcArrow    = 32512
	colorWindow = 5

	vkReturn   = 0x0D
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12 // Alt
	vkEscape   = 0x1B
	vkLWin     = 0x5B
	vkRWin     = 0x5C
	vkLShift   = 0xA0
	vkRShift   = 0xA1
	vkLControl = 0xA2
	vkRControl = 0xA3
	vkLMenu    = 0xA4
	vkRMenu    = 0xA5

	idSave   = 1
	idCancel = 2
)

type wndClassExW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msgW struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type rect struct{ Left, Top, Right, Bottom int32 }

var (
	classOnce sync.Once
	classErr  error
	className *uint16
)

// winDialog holds the window handles for the open dialog.
type winDialog struct {
	hwnd, combo, hint, saveBtn uintptr
}

func utf16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func registerClass() {
	className = utf16("MicoffHotkeyRecorder")
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	hInst, _, _ := pGetModuleHandleW.Call(0)
	wc := wndClassExW{
		Size:       uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:    syscall.NewCallback(wndProc),
		Instance:   hInst,
		Cursor:     cursor,
		Background: colorWindow + 1,
		ClassName:  className,
	}
	if r, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		classErr = fmt.Errorf("RegisterClassEx: %v", err)
	}
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCommand:
		switch uint16(wParam) {
		case idSave:
			if d := current(); d != nil {
				d.trySave()
			}
		case idCancel:
			pDestroyWindow.Call(hwnd)
		}
		return 0
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

func keyDown(vk uintptr) bool {
	r, _, _ := pGetKeyState.Call(vk)
	return int16(r) < 0
}

func modifiersDown() keys.Spec {
	return keys.Spec{
		Ctrl:  keyDown(vkControl),
		Alt:   keyDown(vkMenu),
		Shift: keyDown(vkShift),
		Cmd:   keyDown(vkLWin) || keyDown(vkRWin),
	}
}

func isModifierKey(vk uint32) bool {
	switch vk {
	case vkShift, vkControl, vkMenu, vkLWin, vkRWin,
		vkLShift, vkRShift, vkLControl, vkRControl, vkLMenu, vkRMenu:
		return true
	}
	return false
}

// Show opens the "Change hotkey" window and blocks until it closes. save is
// called with the captured combination when the user confirms; if it
// returns an error the window stays open and shows it.
func Show(cur keys.Spec, save func(keys.Spec) error) error {
	w := &winDialog{}
	d := &dialog{s: New(cur, "windows"), save: save}
	d.render = func(v View) {
		pSetWindowTextW.Call(w.combo, uintptr(unsafe.Pointer(utf16(v.Combo))))
		pSetWindowTextW.Call(w.hint, uintptr(unsafe.Pointer(utf16(v.Hint))))
		enable := uintptr(0)
		if v.CanSave {
			enable = 1
		}
		pEnableWindow.Call(w.saveBtn, enable)
	}
	d.close = func() { pDestroyWindow.Call(w.hwnd) }
	if !begin(d) {
		return ErrOpen
	}
	defer end()

	// Win32 windows belong to the thread that creates them.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	classOnce.Do(registerClass)
	if classErr != nil {
		return classErr
	}
	hInst, _, _ := pGetModuleHandleW.Call(0)
	const style, exStyle = wsCaption | wsSysMenu, wsExTopmost | wsExDlgModalFrame
	r := rect{0, 0, 420, 190}
	pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), style, 0, exStyle)
	width, height := int(r.Right-r.Left), int(r.Bottom-r.Top)
	sw, _, _ := pGetSystemMetrics.Call(smCXScreen)
	sh, _, _ := pGetSystemMetrics.Call(smCYScreen)
	hwnd, _, err := pCreateWindowExW.Call(exStyle,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16("Change hotkey"))), style,
		uintptr((int(sw)-width)/2), uintptr((int(sh)-height)/2), uintptr(width), uintptr(height),
		0, 0, hInst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx: %v", err)
	}
	w.hwnd = hwnd

	child := func(class, text string, style, x, y, cx, cy, id uintptr) uintptr {
		h, _, _ := pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16(class))), uintptr(unsafe.Pointer(utf16(text))),
			wsChild|wsVisible|style, x, y, cx, cy, hwnd, id, hInst, 0)
		return h
	}
	w.combo = child("STATIC", "", ssCenter, 20, 34, 380, 50, 0)
	w.hint = child("STATIC", "", ssCenter, 20, 92, 380, 40, 0)
	child("BUTTON", "Cancel", wsTabStop|bsPushButton, 222, 146, 90, 30, idCancel)
	w.saveBtn = child("BUTTON", "Save", wsTabStop|bsDefPushButton, 318, 146, 90, 30, idSave)

	bigFont := createFont(-36, 600)
	font := createFont(-16, 400)
	defer pDeleteObject.Call(bigFont)
	defer pDeleteObject.Call(font)
	pSendMessageW.Call(w.combo, wmSetFont, bigFont, 1)
	for _, h := range []uintptr{w.hint, w.saveBtn} {
		pSendMessageW.Call(h, wmSetFont, font, 1)
	}
	d.render(d.s.View())
	pShowWindow.Call(hwnd, swShow)
	pSetForegroundWindow.Call(hwnd)

	// Keyboard messages are handled here, before dispatch, so a key press
	// is captured no matter which control has focus and never activates a
	// button by itself.
	var m msgW
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break // WM_QUIT (posted by WM_DESTROY) or an error
		}
		switch m.Message {
		case wmKeyDown, wmSysKeyDown:
			if m.LParam&(1<<30) != 0 {
				continue // auto-repeat
			}
			vk := uint32(m.WParam)
			if isModifierKey(vk) {
				d.modifiers(modifiersDown())
				continue
			}
			name, ok := hotkey.KeyName(vk)
			switch {
			case ok:
			case vk == vkEscape:
				name = KeyEscape
			case vk == vkReturn:
				name = KeyReturn
			default:
				name = KeyUnsupported
			}
			d.keyDown(name, modifiersDown())
			continue
		case wmKeyUp, wmSysKeyUp:
			if isModifierKey(uint32(m.WParam)) {
				d.modifiers(modifiersDown())
			}
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func createFont(height, weight int) uintptr {
	const cleartype = 5
	h, _, _ := pCreateFontW.Call(uintptr(int32(height)), 0, 0, 0, uintptr(weight),
		0, 0, 0, 1 /*DEFAULT_CHARSET*/, 0, 0, cleartype, 0,
		uintptr(unsafe.Pointer(utf16("Segoe UI"))))
	return h
}
