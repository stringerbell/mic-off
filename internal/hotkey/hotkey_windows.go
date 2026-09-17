//go:build windows

package hotkey

import (
	"fmt"

	gh "golang.design/x/hotkey"

	"micoff/internal/keys"
)

// winKeyCode maps a key name to its Windows virtual-key code.
func winKeyCode(key string) (gh.Key, bool) {
	switch {
	case key == "space":
		return 0x20, true
	case len(key) == 1 && key[0] >= 'a' && key[0] <= 'z':
		return gh.Key(0x41 + key[0] - 'a'), true
	case len(key) == 1 && key[0] >= '0' && key[0] <= '9':
		return gh.Key(0x30 + key[0] - '0'), true
	case keys.IsFunctionKey(key):
		var n int
		fmt.Sscanf(key, "f%d", &n)
		if n >= 1 && n <= 24 {
			return gh.Key(0x70 + n - 1), true
		}
	}
	return 0, false
}

type winRegistrar struct{}

// New returns the Windows registrar (RegisterHotKey; no special permission).
func New() Registrar { return winRegistrar{} }

func (winRegistrar) Register(spec keys.Spec, onPress func()) (func(), error) {
	code, ok := winKeyCode(spec.Key)
	if !ok {
		return nil, fmt.Errorf("key %q is not supported on Windows", spec.Key)
	}
	var mods []gh.Modifier
	if spec.Ctrl {
		mods = append(mods, gh.ModCtrl)
	}
	if spec.Alt {
		mods = append(mods, gh.ModAlt)
	}
	if spec.Shift {
		mods = append(mods, gh.ModShift)
	}
	if spec.Cmd {
		mods = append(mods, gh.ModWin)
	}
	hk := gh.New(mods, code)
	if err := hk.Register(); err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-hk.Keydown():
				onPress()
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
		hk.Unregister()
	}, nil
}
