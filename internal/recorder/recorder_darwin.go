//go:build darwin

package recorder

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdint.h>
#include <stdlib.h>

void micoff_recorder_run(const char *combo, const char *hint, int canSave);
void micoff_recorder_set(const char *combo, const char *hint, int canSave);
void micoff_recorder_close(void);
*/
import "C"

import (
	"unsafe"

	"micoff/internal/hotkey"
	"micoff/internal/keys"
)

// NSEvent modifier flags and the key codes the dialog treats specially.
const (
	flagShift   = 1 << 17
	flagControl = 1 << 18
	flagOption  = 1 << 19
	flagCommand = 1 << 20

	keyCodeReturn      = 36
	keyCodeEscape      = 53
	keyCodeKeypadEnter = 76
)

// Show opens the "Change hotkey" window and blocks until it closes. save is
// called with the captured combination when the user confirms; if it
// returns an error the window stays open and shows it.
func Show(current keys.Spec, save func(keys.Spec) error) error {
	d := &dialog{s: New(current, "darwin"), save: save}
	d.render = func(v View) {
		combo, hint := C.CString(v.Combo), C.CString(v.Hint)
		defer C.free(unsafe.Pointer(combo))
		defer C.free(unsafe.Pointer(hint))
		C.micoff_recorder_set(combo, hint, cBool(v.CanSave))
	}
	d.close = func() { C.micoff_recorder_close() }
	if !begin(d) {
		return ErrOpen
	}
	defer end()
	v := d.s.View()
	combo, hint := C.CString(v.Combo), C.CString(v.Hint)
	defer C.free(unsafe.Pointer(combo))
	defer C.free(unsafe.Pointer(hint))
	C.micoff_recorder_run(combo, hint, cBool(v.CanSave))
	return nil
}

func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func modifiers(flags uint32) keys.Spec {
	return keys.Spec{
		Ctrl:  flags&flagControl != 0,
		Alt:   flags&flagOption != 0,
		Shift: flags&flagShift != 0,
		Cmd:   flags&flagCommand != 0,
	}
}

//export micoffRecorderKeyDown
func micoffRecorderKeyDown(code, flags C.uint32_t) {
	d := current()
	if d == nil {
		return
	}
	name, ok := hotkey.KeyName(uint32(code))
	switch {
	case ok:
	case code == keyCodeEscape:
		name = KeyEscape
	case code == keyCodeReturn || code == keyCodeKeypadEnter:
		name = KeyReturn
	default:
		name = KeyUnsupported
	}
	d.keyDown(name, modifiers(uint32(flags)))
}

//export micoffRecorderFlags
func micoffRecorderFlags(flags C.uint32_t) {
	if d := current(); d != nil {
		d.modifiers(modifiers(uint32(flags)))
	}
}

//export micoffRecorderSave
func micoffRecorderSave() {
	if d := current(); d != nil {
		d.trySave()
	}
}

//export micoffRecorderCancel
func micoffRecorderCancel() {
	if d := current(); d != nil {
		d.close()
	}
}
