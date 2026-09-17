// Package keys defines the platform-neutral hotkey vocabulary: which
// modifiers and keys can be chosen, and how a combo is written in the
// config file ("ctrl+alt+m") and shown to the user ("Ctrl+Alt+M" / "⌃⌥M").
package keys

import (
	"errors"
	"fmt"
	"strings"
)

// Modifier names, in canonical display order.
const (
	Ctrl  = "ctrl"
	Alt   = "alt"   // Option on macOS
	Shift = "shift"
	Cmd   = "cmd"   // Command on macOS, Win on Windows
)

var Modifiers = []string{Ctrl, Alt, Shift, Cmd}

var modifierAliases = map[string]string{
	"ctrl": Ctrl, "control": Ctrl,
	"alt": Alt, "option": Alt, "opt": Alt,
	"shift": Shift,
	"cmd": Cmd, "command": Cmd, "win": Cmd, "super": Cmd, "meta": Cmd,
}

// Keys lists every selectable non-modifier key in menu order.
var Keys = buildKeys()

func buildKeys() []string {
	var k []string
	for c := 'a'; c <= 'z'; c++ {
		k = append(k, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		k = append(k, string(c))
	}
	for i := 1; i <= 12; i++ {
		k = append(k, fmt.Sprintf("f%d", i))
	}
	return append(k, "space")
}

var keySet = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range Keys {
		m[k] = true
	}
	return m
}()

// Spec is a parsed, validated key combination.
type Spec struct {
	Ctrl, Alt, Shift, Cmd bool
	Key                   string
}

// IsFunctionKey reports whether the key is F1..F12, which may be used
// without any modifier.
func IsFunctionKey(key string) bool {
	return len(key) >= 2 && key[0] == 'f' && keySet[key]
}

func (s Spec) HasModifier() bool { return s.Ctrl || s.Alt || s.Shift || s.Cmd }

func (s Spec) Modifier(name string) bool {
	switch name {
	case Ctrl:
		return s.Ctrl
	case Alt:
		return s.Alt
	case Shift:
		return s.Shift
	case Cmd:
		return s.Cmd
	}
	return false
}

// WithModifier returns a copy with the named modifier set on or off.
func (s Spec) WithModifier(name string, on bool) Spec {
	switch name {
	case Ctrl:
		s.Ctrl = on
	case Alt:
		s.Alt = on
	case Shift:
		s.Shift = on
	case Cmd:
		s.Cmd = on
	}
	return s
}

// WithKey returns a copy with a different key.
func (s Spec) WithKey(key string) Spec {
	s.Key = key
	return s
}

// Validate rejects combos that are unusable: an unknown key, or a plain
// letter/digit with no modifier (which would eat ordinary typing).
func (s Spec) Validate() error {
	if !keySet[s.Key] {
		return fmt.Errorf("unknown key %q", s.Key)
	}
	if !s.HasModifier() && !IsFunctionKey(s.Key) {
		return errors.New("choose at least one modifier (Ctrl, Alt, Shift or Cmd) unless the key is F1-F12")
	}
	return nil
}

// String renders the canonical config form, e.g. "ctrl+alt+m".
func (s Spec) String() string {
	var parts []string
	for _, m := range Modifiers {
		if s.Modifier(m) {
			parts = append(parts, m)
		}
	}
	return strings.Join(append(parts, s.Key), "+")
}

// Parse reads a combo like "ctrl+alt+m", "Cmd+Shift+F5" or "F13".
func Parse(text string) (Spec, error) {
	var s Spec
	parts := strings.Split(strings.ToLower(strings.TrimSpace(text)), "+")
	if len(parts) == 0 || text == "" {
		return s, errors.New("empty hotkey")
	}
	for i, p := range parts {
		p = strings.TrimSpace(p)
		last := i == len(parts)-1
		if canon, ok := modifierAliases[p]; ok && !last {
			s = s.WithModifier(canon, true)
			continue
		}
		if !last {
			return s, fmt.Errorf("hotkey %q: %q is not a modifier", text, p)
		}
		s.Key = p
	}
	if err := s.Validate(); err != nil {
		return s, fmt.Errorf("hotkey %q: %w", text, err)
	}
	return s, nil
}

// KeyLabel is the human name of a key: "M", "5", "F12", "Space".
func KeyLabel(key string) string {
	if key == "space" {
		return "Space"
	}
	return strings.ToUpper(key)
}

// ModifierLabel is the human name of a modifier on the given platform.
func ModifierLabel(mod, platform string) string {
	mac := platform == "darwin"
	switch mod {
	case Ctrl:
		if mac {
			return "Control ⌃"
		}
		return "Ctrl"
	case Alt:
		if mac {
			return "Option ⌥"
		}
		return "Alt"
	case Shift:
		if mac {
			return "Shift ⇧"
		}
		return "Shift"
	case Cmd:
		if mac {
			return "Command ⌘"
		}
		return "Win"
	}
	return mod
}

// Display renders the combo the way the platform's users expect:
// "⌃⌥M" on macOS, "Ctrl+Alt+M" elsewhere.
func Display(s Spec, platform string) string {
	if platform == "darwin" {
		var b strings.Builder
		if s.Ctrl {
			b.WriteString("⌃")
		}
		if s.Alt {
			b.WriteString("⌥")
		}
		if s.Shift {
			b.WriteString("⇧")
		}
		if s.Cmd {
			b.WriteString("⌘")
		}
		b.WriteString(KeyLabel(s.Key))
		return b.String()
	}
	var parts []string
	for _, m := range Modifiers {
		if s.Modifier(m) {
			parts = append(parts, ModifierLabel(m, platform))
		}
	}
	return strings.Join(append(parts, KeyLabel(s.Key)), "+")
}
