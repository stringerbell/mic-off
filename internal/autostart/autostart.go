// Package autostart registers mic-off to launch when the user logs in.
package autostart

import (
	"os"
	"path/filepath"
)

// Autostart enables or disables launch-at-login for the running executable.
type Autostart struct{}

func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}
