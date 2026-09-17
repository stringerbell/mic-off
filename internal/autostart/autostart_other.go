//go:build !darwin && !windows

package autostart

import "errors"

func (Autostart) Enable(bool) error {
	return errors.New("start at login is supported on macOS and Windows only")
}
