// Package app is the platform-independent brain of mic-off: it owns the
// current hotkey, drives the microphone, persists settings, and tells the UI
// when something changed. Everything OS-specific is injected.
package app

import (
	"fmt"
	"sync"

	"micoff/internal/config"
	"micoff/internal/hotkey"
	"micoff/internal/keys"
	"micoff/internal/mic"
)

// Autostart toggles "launch at login".
type Autostart interface {
	Enable(on bool) error
}

// Deps are the collaborators the Controller needs.
type Deps struct {
	Mic       mic.Mic
	Hotkeys   hotkey.Registrar
	Autostart Autostart
	Config    config.Config
	Save      func(config.Config) error
	Logf      func(format string, args ...any)
	// OnState is called whenever the mute state changes (from the hotkey,
	// the menu, or another program). Called without locks held.
	OnState func(muted bool)
	// OnHotkey is called whenever the active hotkey or its registration
	// status changes. err is non-nil when the combo could not be registered.
	OnHotkey func(spec keys.Spec, err error)
}

type Controller struct {
	d Deps

	mu         sync.Mutex
	cfg        config.Config
	spec       keys.Spec
	unregister func()
	hotkeyErr  error
	muted      bool
	known      bool // muted has been read from the device at least once
}

func New(d Deps) *Controller {
	if d.Logf == nil {
		d.Logf = func(string, ...any) {}
	}
	if d.OnState == nil {
		d.OnState = func(bool) {}
	}
	if d.OnHotkey == nil {
		d.OnHotkey = func(keys.Spec, error) {}
	}
	if d.Save == nil {
		d.Save = func(config.Config) error { return nil }
	}
	return &Controller{d: d, cfg: d.Config}
}

// Start parses the configured hotkey (falling back to the default if it is
// unusable), registers it and reads the initial mute state. A hotkey that
// fails to register is reported through OnHotkey but does not stop the app:
// the menu still works.
func (c *Controller) Start() {
	spec, err := keys.Parse(c.cfg.Hotkey)
	if err != nil {
		c.d.Logf("config hotkey unusable (%v); using default %s", err, config.DefaultHotkey)
		spec, _ = keys.Parse(config.DefaultHotkey)
	}
	c.mu.Lock()
	c.spec = spec
	c.cfg.Hotkey = spec.String()
	c.register(spec)
	spec, hkErr := c.spec, c.hotkeyErr
	c.mu.Unlock()
	c.d.OnHotkey(spec, hkErr)
	c.Refresh()
}

// register binds spec and records the outcome. Caller holds c.mu.
func (c *Controller) register(spec keys.Spec) {
	if c.unregister != nil {
		c.unregister()
		c.unregister = nil
	}
	unreg, err := c.d.Hotkeys.Register(spec, c.onHotkeyPressed)
	c.hotkeyErr = err
	if err != nil {
		c.d.Logf("could not register hotkey %s: %v", spec, err)
		return
	}
	c.unregister = unreg
	c.d.Logf("hotkey %s registered", spec)
}

func (c *Controller) onHotkeyPressed() {
	if _, err := c.Toggle(); err != nil {
		c.d.Logf("toggle failed: %v", err)
	}
}

// Stop releases the hotkey.
func (c *Controller) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.unregister != nil {
		c.unregister()
		c.unregister = nil
	}
}

// Toggle flips the microphone and returns the new state.
func (c *Controller) Toggle() (bool, error) {
	cur, err := c.d.Mic.Muted()
	if err != nil {
		return false, fmt.Errorf("reading microphone state: %w", err)
	}
	return c.SetMuted(!cur)
}

// SetMuted sets the microphone to the requested state.
func (c *Controller) SetMuted(muted bool) (bool, error) {
	if err := c.d.Mic.SetMuted(muted); err != nil {
		return muted, fmt.Errorf("setting microphone: %w", err)
	}
	c.d.Logf("microphone %s", stateWord(muted))
	c.Refresh()
	return muted, nil
}

// Refresh re-reads the device and notifies the UI if the state changed. Call
// it periodically so the icon stays truthful when another program (or the
// OS) changes the mic.
func (c *Controller) Refresh() {
	muted, err := c.d.Mic.Muted()
	if err != nil {
		c.d.Logf("reading microphone state: %v", err)
		return
	}
	c.mu.Lock()
	changed := !c.known || muted != c.muted
	c.muted, c.known = muted, true
	c.mu.Unlock()
	if changed {
		c.d.OnState(muted)
	}
}

// Muted returns the last known state without touching the device.
func (c *Controller) Muted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.muted
}

func (c *Controller) Spec() keys.Spec {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.spec
}

func (c *Controller) HotkeyError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hotkeyErr
}

func (c *Controller) Config() config.Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg
}

// SetModifier turns one modifier on or off in the current combo.
func (c *Controller) SetModifier(name string, on bool) error {
	return c.SetSpec(c.Spec().WithModifier(name, on))
}

// SetKey changes the non-modifier key of the current combo.
func (c *Controller) SetKey(key string) error {
	return c.SetSpec(c.Spec().WithKey(key))
}

// SetSpec switches to a new combo. If the new combo cannot be registered the
// previous one is restored and the error returned; nothing is saved.
func (c *Controller) SetSpec(spec keys.Spec) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	old := c.spec
	if spec == old && c.hotkeyErr == nil {
		c.mu.Unlock()
		return nil
	}
	c.register(spec)
	if c.hotkeyErr != nil {
		err := fmt.Errorf("%s is not available: %w", keys.Display(spec, ""), c.hotkeyErr)
		c.register(old) // best effort: put the previous combo back
		spec, hkErr := c.spec, c.hotkeyErr
		c.mu.Unlock()
		c.d.OnHotkey(spec, hkErr)
		return err
	}
	c.spec = spec
	c.cfg.Hotkey = spec.String()
	cfg := c.cfg
	c.mu.Unlock()
	c.d.OnHotkey(spec, nil)
	if err := c.d.Save(cfg); err != nil {
		c.d.Logf("saving config: %v", err)
		return fmt.Errorf("hotkey changed but could not be saved: %w", err)
	}
	return nil
}

// SetStartAtLogin enables or disables launching at login and saves it.
func (c *Controller) SetStartAtLogin(on bool) error {
	if c.d.Autostart != nil {
		if err := c.d.Autostart.Enable(on); err != nil {
			return err
		}
	}
	c.mu.Lock()
	c.cfg.StartAtLogin = on
	cfg := c.cfg
	c.mu.Unlock()
	return c.d.Save(cfg)
}

func stateWord(muted bool) string {
	if muted {
		return "muted"
	}
	return "live"
}
