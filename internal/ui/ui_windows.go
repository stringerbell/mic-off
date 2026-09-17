//go:build windows

package ui

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
)

// Alert shows a modal error dialog.
func Alert(title, message string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	const mbOK, mbIconWarning, mbTopmost = 0x0, 0x30, 0x40000
	messageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbOK|mbIconWarning|mbTopmost)
}

// RevealFile opens Explorer with the file selected.
func RevealFile(path string) error {
	return exec.Command("explorer", "/select,"+path).Start()
}

// HideDockIcon is a no-op on Windows.
func HideDockIcon() {}
