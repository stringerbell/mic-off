//go:build windows

package hotkey

import (
	"testing"

	"micoff/internal/keys"
)

func TestEverySelectableKeyHasWindowsCode(t *testing.T) {
	for _, k := range keys.Keys {
		if _, ok := winKeyCode(k); !ok {
			t.Errorf("key %q has no Windows virtual-key code", k)
		}
	}
	if c, _ := winKeyCode("m"); c != 0x4D {
		t.Errorf("m = %#x", c)
	}
	if c, _ := winKeyCode("f12"); c != 0x7B {
		t.Errorf("f12 = %#x", c)
	}
	if c, _ := winKeyCode("0"); c != 0x30 {
		t.Errorf("0 = %#x", c)
	}
}

func TestKeyNameRoundTrips(t *testing.T) {
	for _, k := range keys.Keys {
		code, _ := winKeyCode(k)
		name, ok := KeyName(uint32(code))
		if !ok || name != k {
			t.Errorf("KeyName(%#x) = %q, %v; want %q", code, name, ok, k)
		}
	}
	if _, ok := KeyName(0x1B); ok { // Escape is not a hotkey key
		t.Error("escape should be unknown")
	}
	if _, ok := KeyName(0x7C); ok { // F13 is outside the supported range
		t.Error("f13 should be unknown")
	}
}
