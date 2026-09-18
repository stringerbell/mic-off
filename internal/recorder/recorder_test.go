package recorder

import (
	"errors"
	"strings"
	"testing"

	"micoff/internal/keys"
)

type fakeWindow struct {
	views  []View
	closed bool
	saved  []keys.Spec
	err    error
}

func newFakeDialog() (*dialog, *fakeWindow) {
	w := &fakeWindow{}
	d := &dialog{
		s:      New(keys.Spec{Ctrl: true, Alt: true, Key: "m"}, "darwin"),
		save:   func(s keys.Spec) error { w.saved = append(w.saved, s); return w.err },
		render: func(v View) { w.views = append(w.views, v) },
		close:  func() { w.closed = true },
	}
	return d, w
}

func TestSaveButtonWithNothingCapturedDoesNothing(t *testing.T) {
	d, w := newFakeDialog()
	d.trySave()
	if len(w.saved) != 0 || w.closed {
		t.Fatalf("saved %v closed %v", w.saved, w.closed)
	}
}

func TestSaveClosesWindowOnSuccess(t *testing.T) {
	d, w := newFakeDialog()
	d.keyDown("k", keys.Spec{Ctrl: true})
	d.trySave()
	if len(w.saved) != 1 || w.saved[0].String() != "ctrl+k" || !w.closed {
		t.Fatalf("saved %v closed %v", w.saved, w.closed)
	}
}

func TestRefusedSaveKeepsWindowOpenAndShowsReason(t *testing.T) {
	d, w := newFakeDialog()
	w.err = errors.New("taken by the system")
	d.keyDown("k", keys.Spec{Ctrl: true})
	d.trySave()
	if w.closed {
		t.Fatal("window must stay open")
	}
	last := w.views[len(w.views)-1]
	if last.CanSave || !strings.Contains(last.Hint, "taken by the system") {
		t.Fatalf("%+v", last)
	}
	// Return must not retry the refused combo.
	d.keyDown(KeyReturn, keys.Spec{})
	if len(w.saved) != 1 {
		t.Fatalf("saved again: %v", w.saved)
	}
}

func TestReturnSavesAndEscapeCloses(t *testing.T) {
	d, w := newFakeDialog()
	d.keyDown("f5", keys.Spec{})
	d.keyDown(KeyReturn, keys.Spec{})
	if len(w.saved) != 1 || !w.closed {
		t.Fatalf("saved %v closed %v", w.saved, w.closed)
	}
	d, w = newFakeDialog()
	d.keyDown(KeyEscape, keys.Spec{})
	if len(w.saved) != 0 || !w.closed {
		t.Fatalf("saved %v closed %v", w.saved, w.closed)
	}
}

func TestOnlyOneDialogAtATime(t *testing.T) {
	d, _ := newFakeDialog()
	if !begin(d) {
		t.Fatal("first dialog should open")
	}
	if begin(d) {
		t.Fatal("second dialog must be refused")
	}
	if current() != d {
		t.Fatal("current should be the open dialog")
	}
	end()
	if current() != nil {
		t.Fatal("nothing should be open")
	}
}
