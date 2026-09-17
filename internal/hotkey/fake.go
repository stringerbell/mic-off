package hotkey

import (
	"errors"
	"micoff/internal/keys"
)

// Fake records registrations for tests and lets them be triggered.
type Fake struct {
	Active   map[string]func() // currently registered combos → callback
	Rejected map[string]bool   // combos Register refuses
	Log      []string          // "+combo" on register, "-combo" on unregister
}

func NewFake() *Fake {
	return &Fake{Active: map[string]func(){}, Rejected: map[string]bool{}}
}

func (f *Fake) Register(spec keys.Spec, onPress func()) (func(), error) {
	name := spec.String()
	if f.Rejected[name] {
		return nil, errors.New("fake: combo rejected")
	}
	if _, dup := f.Active[name]; dup {
		return nil, errors.New("fake: already registered")
	}
	f.Active[name] = onPress
	f.Log = append(f.Log, "+"+name)
	return func() {
		delete(f.Active, name)
		f.Log = append(f.Log, "-"+name)
	}, nil
}

// Press simulates the user pressing a registered combo.
func (f *Fake) Press(combo string) bool {
	cb, ok := f.Active[combo]
	if ok {
		cb()
	}
	return ok
}
