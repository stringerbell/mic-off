package app

import (
	"errors"
	"testing"

	"micoff/internal/config"
	"micoff/internal/hotkey"
	"micoff/internal/keys"
	"micoff/internal/mic"
)

type harness struct {
	c       *Controller
	mic     *mic.Fake
	hk      *hotkey.Fake
	saved   []config.Config
	states  []bool
	hotkeys []string
	hkErrs  []error
	auto    []bool
	autoErr error
}

func (h *harness) Enable(on bool) error {
	h.auto = append(h.auto, on)
	return h.autoErr
}

func newHarness(t *testing.T, cfg config.Config) *harness {
	t.Helper()
	h := &harness{mic: &mic.Fake{}, hk: hotkey.NewFake()}
	h.c = New(Deps{
		Mic:       h.mic,
		Hotkeys:   h.hk,
		Autostart: h,
		Config:    cfg,
		Save:      func(c config.Config) error { h.saved = append(h.saved, c); return nil },
		Logf:      t.Logf,
		OnState:   func(m bool) { h.states = append(h.states, m) },
		OnHotkey: func(s keys.Spec, err error) {
			h.hotkeys = append(h.hotkeys, s.String())
			h.hkErrs = append(h.hkErrs, err)
		},
	})
	return h
}

func TestStartRegistersConfiguredHotkeyAndReportsInitialState(t *testing.T) {
	h := newHarness(t, config.Config{Hotkey: "cmd+shift+k"}) // canonical form is shift+cmd+k
	h.c.Start()
	if _, ok := h.hk.Active["shift+cmd+k"]; !ok {
		t.Fatalf("hotkey not registered: %v", h.hk.Log)
	}
	if len(h.states) != 1 || h.states[0] != false {
		t.Fatalf("initial state notification: %v", h.states)
	}
	if h.c.HotkeyError() != nil {
		t.Fatal(h.c.HotkeyError())
	}
}

func TestHotkeyPressTogglesMicAndNotifies(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	if !h.hk.Press("ctrl+alt+m") {
		t.Fatal("default combo not active")
	}
	if m, _ := h.mic.Muted(); !m {
		t.Fatal("mic should be muted after first press")
	}
	h.hk.Press("ctrl+alt+m")
	if m, _ := h.mic.Muted(); m {
		t.Fatal("mic should be live after second press")
	}
	want := []bool{false, true, false}
	if len(h.states) != len(want) {
		t.Fatalf("states %v want %v", h.states, want)
	}
	for i := range want {
		if h.states[i] != want[i] {
			t.Fatalf("states %v want %v", h.states, want)
		}
	}
}

func TestInvalidConfiguredHotkeyFallsBackToDefault(t *testing.T) {
	h := newHarness(t, config.Config{Hotkey: "m"}) // bare letter is rejected
	h.c.Start()
	if _, ok := h.hk.Active[config.DefaultHotkey]; !ok {
		t.Fatalf("default not registered: %v", h.hk.Log)
	}
	if h.c.Config().Hotkey != config.DefaultHotkey {
		t.Fatalf("config hotkey %q", h.c.Config().Hotkey)
	}
}

func TestSetKeyRebindsAndSaves(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	if err := h.c.SetKey("k"); err != nil {
		t.Fatal(err)
	}
	if _, old := h.hk.Active["ctrl+alt+m"]; old {
		t.Fatal("old combo still registered")
	}
	if !h.hk.Press("ctrl+alt+k") {
		t.Fatal("new combo not active")
	}
	if len(h.saved) != 1 || h.saved[0].Hotkey != "ctrl+alt+k" {
		t.Fatalf("saved: %+v", h.saved)
	}
	if last := h.hotkeys[len(h.hotkeys)-1]; last != "ctrl+alt+k" {
		t.Fatalf("OnHotkey got %q", last)
	}
}

func TestSetModifierAddsAndRemoves(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	if err := h.c.SetModifier(keys.Shift, true); err != nil {
		t.Fatal(err)
	}
	if err := h.c.SetModifier(keys.Alt, false); err != nil {
		t.Fatal(err)
	}
	if got := h.c.Spec().String(); got != "ctrl+shift+m" {
		t.Fatalf("spec %q", got)
	}
	if !h.hk.Press("ctrl+shift+m") {
		t.Fatal("combo not active")
	}
}

func TestRemovingLastModifierFromLetterIsRefused(t *testing.T) {
	h := newHarness(t, config.Config{Hotkey: "ctrl+m"})
	h.c.Start()
	if err := h.c.SetModifier(keys.Ctrl, false); err == nil {
		t.Fatal("expected validation error")
	}
	if !h.hk.Press("ctrl+m") {
		t.Fatal("original combo should remain active")
	}
	if len(h.saved) != 0 {
		t.Fatalf("nothing should be saved: %+v", h.saved)
	}
}

func TestFunctionKeyAloneIsAllowed(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	if err := h.c.SetSpec(keys.Spec{Key: "f9"}); err != nil {
		t.Fatal(err)
	}
	if !h.hk.Press("f9") {
		t.Fatal("f9 not active")
	}
}

func TestRegistrationFailureRestoresPreviousCombo(t *testing.T) {
	h := newHarness(t, config.Default())
	h.hk.Rejected["ctrl+alt+z"] = true
	h.c.Start()
	err := h.c.SetKey("z")
	if err == nil {
		t.Fatal("expected error")
	}
	if !h.hk.Press("ctrl+alt+m") {
		t.Fatalf("previous combo not restored: %v", h.hk.Log)
	}
	if h.c.Spec().String() != "ctrl+alt+m" || h.c.HotkeyError() != nil {
		t.Fatalf("spec %s err %v", h.c.Spec(), h.c.HotkeyError())
	}
	if len(h.saved) != 0 {
		t.Fatalf("failed change must not be saved: %+v", h.saved)
	}
	// Only one "-ctrl+alt+m" then "+ctrl+alt+m" round trip; no leaks.
	if len(h.hk.Active) != 1 {
		t.Fatalf("active combos: %v", h.hk.Active)
	}
}

func TestStartWithUnregistrableHotkeyStillAllowsMenuToggle(t *testing.T) {
	h := newHarness(t, config.Default())
	h.hk.Rejected[config.DefaultHotkey] = true
	h.c.Start()
	if h.c.HotkeyError() == nil {
		t.Fatal("expected hotkey error to be recorded")
	}
	if h.hkErrs[0] == nil {
		t.Fatal("OnHotkey should carry the error")
	}
	if m, err := h.c.Toggle(); err != nil || !m {
		t.Fatalf("menu toggle: muted=%v err=%v", m, err)
	}
	// Picking a working combo clears the error.
	if err := h.c.SetKey("k"); err != nil {
		t.Fatal(err)
	}
	if h.c.HotkeyError() != nil {
		t.Fatal("error should be cleared")
	}
}

func TestRefreshDetectsExternalChange(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	h.mic.SetExternally(true)
	h.c.Refresh()
	h.c.Refresh() // no duplicate notification for the same state
	if len(h.states) != 2 || !h.states[1] {
		t.Fatalf("states %v", h.states)
	}
	if !h.c.Muted() {
		t.Fatal("cached state not updated")
	}
}

func TestMicErrorSurfacesAndStateUnchanged(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	h.mic.Err = errors.New("device gone")
	if _, err := h.c.Toggle(); err == nil {
		t.Fatal("expected error")
	}
	if len(h.mic.Calls) != 0 {
		t.Fatal("SetMuted should not have been attempted")
	}
	if len(h.states) != 1 {
		t.Fatalf("no new state should be reported: %v", h.states)
	}
}

func TestStartAtLoginPersistsAndCallsAutostart(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	if err := h.c.SetStartAtLogin(true); err != nil {
		t.Fatal(err)
	}
	if len(h.auto) != 1 || !h.auto[0] {
		t.Fatalf("autostart calls %v", h.auto)
	}
	if len(h.saved) != 1 || !h.saved[0].StartAtLogin || h.saved[0].Hotkey != config.DefaultHotkey {
		t.Fatalf("saved %+v", h.saved)
	}
	h.autoErr = errors.New("no permission")
	if err := h.c.SetStartAtLogin(false); err == nil {
		t.Fatal("expected error")
	}
	if len(h.saved) != 1 {
		t.Fatal("config must not be saved when autostart fails")
	}
}

func TestStopReleasesHotkey(t *testing.T) {
	h := newHarness(t, config.Default())
	h.c.Start()
	h.c.Stop()
	if len(h.hk.Active) != 0 {
		t.Fatalf("still active: %v", h.hk.Active)
	}
}
