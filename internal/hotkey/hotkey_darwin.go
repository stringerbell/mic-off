//go:build darwin

package hotkey

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#include <stdint.h>
#include <Carbon/Carbon.h>

int micoff_register(uint32_t key, uint32_t mods, uint32_t id, EventHotKeyRef *ref);
int micoff_unregister(EventHotKeyRef ref);
*/
import "C"

import (
	"fmt"
	"sync"

	"micoff/internal/keys"
)

// Carbon modifier masks (Events.h).
const (
	carbonCmd   = 0x100
	carbonShift = 0x200
	carbonAlt   = 0x800
	carbonCtrl  = 0x1000
)

// Carbon virtual key codes (kVK_* in Events.h).
var macKeyCodes = map[string]uint32{
	"a": 0, "s": 1, "d": 2, "f": 3, "h": 4, "g": 5, "z": 6, "x": 7, "c": 8, "v": 9,
	"b": 11, "q": 12, "w": 13, "e": 14, "r": 15, "y": 16, "t": 17,
	"1": 18, "2": 19, "3": 20, "4": 21, "6": 22, "5": 23, "9": 25, "7": 26, "8": 28, "0": 29,
	"o": 31, "u": 32, "i": 34, "p": 35, "l": 37, "j": 38, "k": 40, "n": 45, "m": 46,
	"space": 49,
	"f1":    0x7A, "f2": 0x78, "f3": 0x63, "f4": 0x76, "f5": 0x60, "f6": 0x61,
	"f7": 0x62, "f8": 0x64, "f9": 0x65, "f10": 0x6D, "f11": 0x67, "f12": 0x6F,
}

var (
	mu        sync.Mutex
	callbacks = map[uint32]func(){}
	nextID    uint32
)

//export micoffHotkeyPressed
func micoffHotkeyPressed(id uint32) {
	mu.Lock()
	cb := callbacks[id]
	mu.Unlock()
	if cb != nil {
		go cb() // never block the Carbon event handler
	}
}

type carbonRegistrar struct{}

// New returns the macOS registrar. It uses Carbon's RegisterEventHotKey,
// which needs no Accessibility or Input Monitoring permission. The process
// must be running a Cocoa main run loop (the tray does that).
func New() Registrar { return carbonRegistrar{} }

func (carbonRegistrar) Register(spec keys.Spec, onPress func()) (func(), error) {
	code, ok := macKeyCodes[spec.Key]
	if !ok {
		return nil, fmt.Errorf("key %q is not supported on macOS", spec.Key)
	}
	var mods uint32
	if spec.Ctrl {
		mods |= carbonCtrl
	}
	if spec.Alt {
		mods |= carbonAlt
	}
	if spec.Shift {
		mods |= carbonShift
	}
	if spec.Cmd {
		mods |= carbonCmd
	}

	mu.Lock()
	nextID++
	id := nextID
	callbacks[id] = onPress
	mu.Unlock()

	var ref C.EventHotKeyRef
	if st := C.micoff_register(C.uint32_t(code), C.uint32_t(mods), C.uint32_t(id), &ref); st != 0 {
		mu.Lock()
		delete(callbacks, id)
		mu.Unlock()
		if int(st) == -9878 { // eventHotKeyExistsErr
			return nil, fmt.Errorf("already in use by the system or another app")
		}
		return nil, fmt.Errorf("RegisterEventHotKey failed (status %d)", int(st))
	}
	return func() {
		C.micoff_unregister(ref)
		mu.Lock()
		delete(callbacks, id)
		mu.Unlock()
	}, nil
}

// KeyName maps a Carbon virtual key code back to its keys name, so the
// recorder shows exactly the key that Register would bind.
func KeyName(code uint32) (string, bool) {
	name, ok := macKeyNames[code]
	return name, ok
}

var macKeyNames = func() map[uint32]string {
	m := map[uint32]string{}
	for name, code := range macKeyCodes {
		m[code] = name
	}
	return m
}()
