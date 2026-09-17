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
