// Package hotkey registers a system-wide key combination.
package hotkey

import "micoff/internal/keys"

// Registrar binds a combo to a callback. Unregister releases it.
type Registrar interface {
	Register(spec keys.Spec, onPress func()) (unregister func(), err error)
}
