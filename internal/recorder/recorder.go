package recorder

import (
	"errors"
	"sync"

	"micoff/internal/keys"
)

// ErrOpen is returned by Show while another dialog is already open.
var ErrOpen = errors.New("the hotkey window is already open")

// dialog is one open native window. The platform files create the window,
// feed keyboard events to keyDown/modifiers, and supply render and close.
type dialog struct {
	s      *Session
	save   func(keys.Spec) error
	render func(View)
	close  func()
}

var (
	mu     sync.Mutex
	active *dialog
)

// begin claims the single dialog slot; false if one is open.
func begin(d *dialog) bool {
	mu.Lock()
	defer mu.Unlock()
	if active != nil {
		return false
	}
	active = d
	return true
}

func end() {
	mu.Lock()
	active = nil
	mu.Unlock()
}

// current is the open dialog, or nil. Native callbacks use it so a stray
// event after the window closed is ignored.
func current() *dialog {
	mu.Lock()
	defer mu.Unlock()
	return active
}

// keyDown and modifiers take the held modifiers as a Spec (Key unused).
func (d *dialog) keyDown(name string, m keys.Spec) {
	v, act := d.s.KeyDown(name, m.Ctrl, m.Alt, m.Shift, m.Cmd)
	switch act {
	case Save:
		d.trySave()
	case Cancel:
		d.close()
	default:
		d.render(v)
	}
}

func (d *dialog) modifiers(m keys.Spec) {
	d.render(d.s.Modifiers(m.Ctrl, m.Alt, m.Shift, m.Cmd))
}

// trySave is the Save button. Only a combo the Session accepts reaches the
// save callback, and a callback error keeps the window open with the reason.
func (d *dialog) trySave() {
	spec, ok := d.s.Spec()
	if !ok {
		d.render(d.s.View())
		return
	}
	if err := d.save(spec); err != nil {
		d.render(d.s.Fail(err))
		return
	}
	d.close()
}
