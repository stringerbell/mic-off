//go:build !darwin && !windows

package hotkey

import (
	"errors"

	"micoff/internal/keys"
)

type unsupported struct{}

func New() Registrar { return unsupported{} }

func (unsupported) Register(keys.Spec, func()) (func(), error) {
	return nil, errors.New("global hotkeys are supported on macOS and Windows only")
}
