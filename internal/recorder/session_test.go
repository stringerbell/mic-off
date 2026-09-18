package recorder

import (
	"errors"
	"strings"
	"testing"

	"micoff/internal/keys"
)

func newMac() *Session {
	return New(keys.Spec{Ctrl: true, Alt: true, Key: "m"}, "darwin")
}

func TestInitialViewShowsCurrentComboAndCannotSave(t *testing.T) {
	v := newMac().View()
	if v.Combo != "⌃⌥M" || v.CanSave {
		t.Fatalf("%+v", v)
	}
	if v.Hint == "" {
		t.Fatal("expected an instruction")
	}
}

func TestHeldModifiersAreShownLiveThenRevert(t *testing.T) {
	s := newMac()
	v := s.Modifiers(true, false, false, false)
	if v.Combo != "⌃" || v.CanSave {
		t.Fatalf("holding ctrl: %+v", v)
	}
	v = s.Modifiers(true, false, false, true)
	if v.Combo != "⌃⌘" {
		t.Fatalf("holding ctrl+cmd: %+v", v)
	}
	v = s.Modifiers(false, false, false, false)
	if v.Combo != "⌃⌥M" {
		t.Fatalf("released without a key: %+v", v)
	}
	if _, ok := s.Spec(); ok {
		t.Fatal("nothing was captured")
	}
}

func TestLetterWithModifiersIsCaptured(t *testing.T) {
	s := newMac()
	s.Modifiers(false, false, true, true)
	v, act := s.KeyDown("k", false, false, true, true)
	if act != Continue || !v.CanSave || v.Combo != "⇧⌘K" {
		t.Fatalf("%+v %v", v, act)
	}
	spec, ok := s.Spec()
	if !ok || spec.String() != "shift+cmd+k" {
		t.Fatalf("spec %v ok %v", spec, ok)
	}
	// Letting go of the modifiers one at a time keeps showing the capture.
	if v := s.Modifiers(false, false, false, true); v.Combo != "⇧⌘K" || !v.CanSave {
		t.Fatalf("after releasing shift: %+v", v)
	}
	if v := s.Modifiers(false, false, false, false); v.Combo != "⇧⌘K" || !v.CanSave {
		t.Fatalf("after releasing all: %+v", v)
	}
}

func TestPlainLetterCannotBeSaved(t *testing.T) {
	s := newMac()
	v, _ := s.KeyDown("m", false, false, false, false)
	if v.CanSave || v.Combo != "M" {
		t.Fatalf("%+v", v)
	}
	if !strings.Contains(v.Hint, "Control, Option, Shift or Command") {
		t.Fatalf("hint should name the mac modifiers: %q", v.Hint)
	}
	if _, ok := s.Spec(); ok {
		t.Fatal("must not be saveable")
	}
	// Return does nothing while the combo is unusable.
	if _, act := s.KeyDown(KeyReturn, false, false, false, false); act != Continue {
		t.Fatalf("return acted: %v", act)
	}
}

func TestWindowsWordingNamesWinKey(t *testing.T) {
	s := New(keys.Spec{Ctrl: true, Alt: true, Key: "m"}, "windows")
	if v := s.View(); v.Combo != "Ctrl+Alt+M" {
		t.Fatalf("%+v", v)
	}
	v, _ := s.KeyDown("5", false, false, false, false)
	if !strings.Contains(v.Hint, "Ctrl, Alt, Shift or Win") {
		t.Fatalf("hint: %q", v.Hint)
	}
	if v := s.Modifiers(true, true, false, false); v.Combo != "Ctrl+Alt" {
		t.Fatalf("held: %+v", v)
	}
}

func TestFunctionKeyAloneIsSaveable(t *testing.T) {
	s := newMac()
	v, _ := s.KeyDown("f9", false, false, false, false)
	if !v.CanSave || v.Combo != "F9" {
		t.Fatalf("%+v", v)
	}
	if _, act := s.KeyDown(KeyReturn, false, false, false, false); act != Save {
		t.Fatalf("return should save: %v", act)
	}
}

func TestUnsupportedKeyReplacesEarlierCapture(t *testing.T) {
	s := newMac()
	s.KeyDown("k", true, false, false, false)
	v, act := s.KeyDown(KeyUnsupported, true, false, false, false)
	if act != Continue || v.CanSave || v.Combo != "⌃?" {
		t.Fatalf("%+v %v", v, act)
	}
	if _, ok := s.Spec(); ok {
		t.Fatal("the earlier capture must be forgotten")
	}
	// Escape/Return with a modifier are ordinary unusable keys, not actions.
	if v, act := s.KeyDown(KeyEscape, false, true, false, false); act != Continue || v.Combo != "⌥?" {
		t.Fatalf("%+v %v", v, act)
	}
	if v, act := s.KeyDown(KeyReturn, false, false, false, true); act != Continue || v.Combo != "⌘?" {
		t.Fatalf("%+v %v", v, act)
	}
}

func TestEscapeCancels(t *testing.T) {
	s := newMac()
	s.KeyDown("k", true, false, false, false)
	if _, act := s.KeyDown(KeyEscape, false, false, false, false); act != Cancel {
		t.Fatalf("%v", act)
	}
}

func TestFailedSaveKeepsDialogOpenUntilNewCombo(t *testing.T) {
	s := newMac()
	s.KeyDown("z", true, true, false, false)
	v := s.Fail(errors.New("already taken"))
	if v.CanSave || !strings.Contains(v.Hint, "already taken") || v.Combo != "⌃⌥Z" {
		t.Fatalf("%+v", v)
	}
	if _, ok := s.Spec(); ok {
		t.Fatal("a refused combo must not be saveable")
	}
	if _, act := s.KeyDown(KeyReturn, false, false, false, false); act != Continue {
		t.Fatalf("return after failure acted: %v", act)
	}
	v, _ = s.KeyDown("x", true, true, false, false)
	if !v.CanSave || v.Combo != "⌃⌥X" {
		t.Fatalf("new combo after failure: %+v", v)
	}
}

func TestNewModifierAfterCaptureShowsHeldThenRevertsToCapture(t *testing.T) {
	s := newMac()
	s.KeyDown("k", true, false, false, false)
	s.Modifiers(false, false, false, false)
	if v := s.Modifiers(false, false, true, false); v.Combo != "⇧" || v.CanSave {
		t.Fatalf("holding shift: %+v", v)
	}
	if v := s.Modifiers(false, false, false, false); v.Combo != "⌃K" || !v.CanSave {
		t.Fatalf("released: %+v", v)
	}
}
