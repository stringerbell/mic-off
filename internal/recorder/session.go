// Package recorder is the "press the new key combination" dialog used to
// change the hotkey. Session is the platform-neutral logic behind it: it
// turns raw keyboard events into a candidate combo, decides whether that
// combo may be saved, and produces the text the dialog shows. The
// platform files open a native window, feed it events and render its View.
//
// Nothing in here touches the hotkey registration or the config: the
// dialog only ever hands a validated Spec to the save callback, and if
// saving fails the dialog stays open and shows why. There is no way to end
// up with a half-applied combination.
package recorder

import (
	"strings"

	"micoff/internal/keys"
)

// Escape and Return are the two non-hotkey keys the dialog understands.
// Platform code maps its own key codes to these names.
const (
	KeyEscape = "escape"
	KeyReturn = "return"
	// KeyUnsupported is what platform code passes for a key that can never be
	// a hotkey (arrows, punctuation, ...). It is shown as "?".
	KeyUnsupported = "?"
)

// View is everything the dialog displays.
type View struct {
	Combo   string // the combination, shown large
	Hint    string // one line under it: an instruction, or why Save is disabled
	CanSave bool
}

// Action is what the dialog should do after a key press.
type Action int

const (
	Continue Action = iota
	Save            // Return was pressed with a saveable combo
	Cancel          // Escape was pressed
)

// Session tracks one open dialog.
type Session struct {
	platform string
	current  keys.Spec

	held     keys.Spec // modifiers currently down (Key is always "")
	showHeld bool      // show held modifiers instead of the captured combo
	captured *keys.Spec
	problem  string // why captured cannot be saved; "" when it can
}

// New starts a session showing the current combo. platform is runtime.GOOS
// and only affects wording ("⌃⌥M" vs "Ctrl+Alt+M").
func New(current keys.Spec, platform string) *Session {
	return &Session{platform: platform, current: current}
}

// Modifiers reports that the set of held modifier keys changed.
func (s *Session) Modifiers(ctrl, alt, shift, cmd bool) View {
	now := keys.Spec{Ctrl: ctrl, Alt: alt, Shift: shift, Cmd: cmd}
	switch {
	case (now.Ctrl && !s.held.Ctrl) || (now.Alt && !s.held.Alt) ||
		(now.Shift && !s.held.Shift) || (now.Cmd && !s.held.Cmd):
		s.showHeld = true // a modifier was added: the user is starting a combo
	case !now.HasModifier():
		s.showHeld = false // everything released: back to the result
	}
	// Releasing one of several modifiers after a capture keeps showing the
	// capture, so "⌃⌥M" does not flash to "⌃" while the user lets go.
	s.held = now
	return s.View()
}

// KeyDown reports a (non-repeating) press of a non-modifier key. key is a
// keys name ("m", "f5", "space"), KeyEscape, KeyReturn or KeyUnsupported.
// The modifier flags are those held at the moment of the press.
func (s *Session) KeyDown(key string, ctrl, alt, shift, cmd bool) (View, Action) {
	s.held = keys.Spec{Ctrl: ctrl, Alt: alt, Shift: shift, Cmd: cmd}
	s.showHeld = false
	plain := !s.held.HasModifier()
	switch key {
	case KeyEscape:
		if plain {
			return s.View(), Cancel
		}
	case KeyReturn:
		if plain {
			if s.View().CanSave {
				return s.View(), Save
			}
			return s.View(), Continue
		}
	}
	spec := s.held.WithKey(key)
	s.captured = &spec
	switch {
	case key == KeyEscape || key == KeyReturn || key == KeyUnsupported:
		spec.Key = KeyUnsupported
		s.problem = "That key can't be used. Try a letter, number, F1–F12 or Space."
	case spec.Validate() != nil && !spec.HasModifier():
		s.problem = "Hold " + s.modifierNames() + " while pressing it, or use F1–F12 on its own."
	case spec.Validate() != nil:
		s.problem = spec.Validate().Error()
	default:
		s.problem = ""
	}
	return s.View(), Continue
}

// Fail reports that saving the captured combo was refused (for example the
// OS would not register it). The dialog stays open and shows why.
func (s *Session) Fail(err error) View {
	s.problem = "That combination can't be used: " + err.Error()
	return s.View()
}

// Spec returns the captured combo when it may be saved.
func (s *Session) Spec() (keys.Spec, bool) {
	if s.captured == nil || s.problem != "" {
		return keys.Spec{}, false
	}
	return *s.captured, true
}

// View renders the current state.
func (s *Session) View() View {
	if s.showHeld {
		return View{Combo: keys.Display(s.held, s.platform), Hint: "Now press a key."}
	}
	if s.captured == nil {
		return View{Combo: keys.Display(s.current, s.platform), Hint: "Press the new key combination."}
	}
	if s.problem != "" {
		return View{Combo: keys.Display(*s.captured, s.platform), Hint: s.problem}
	}
	return View{
		Combo:   keys.Display(*s.captured, s.platform),
		Hint:    "Click Save to use it, or press a different combination.",
		CanSave: true,
	}
}

// modifierNames is "Control, Option, Shift or Command" on macOS and
// "Ctrl, Alt, Shift or Win" on Windows.
func (s *Session) modifierNames() string {
	var names []string
	for _, m := range keys.Modifiers {
		label := keys.ModifierLabel(m, s.platform)
		names = append(names, strings.SplitN(label, " ", 2)[0]) // drop the "⌃" symbol suffix
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}
