//go:build !darwin && !windows

package recorder

import (
	"errors"

	"micoff/internal/keys"
)

// Show is unavailable off macOS and Windows.
func Show(keys.Spec, func(keys.Spec) error) error {
	return errors.New("the hotkey window is available on macOS and Windows only")
}
