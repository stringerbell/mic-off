//go:build darwin

package hotkey

import (
	"testing"

	"micoff/internal/keys"
)

// Every key the menu offers must have a Carbon key code, otherwise picking
// it would silently fail to register.
func TestEverySelectableKeyHasMacCode(t *testing.T) {
	for _, k := range keys.Keys {
		if _, ok := macKeyCodes[k]; !ok {
			t.Errorf("key %q has no macOS key code", k)
		}
	}
	seen := map[uint32]string{}
	for name, code := range macKeyCodes {
		if other, dup := seen[code]; dup {
			t.Errorf("key code %d used by both %q and %q", code, name, other)
		}
		seen[code] = name
	}
}
