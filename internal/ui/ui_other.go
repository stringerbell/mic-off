//go:build !darwin && !windows

package ui

import (
	"fmt"
	"os"
	"os/exec"
)

func Alert(title, message string) { fmt.Fprintf(os.Stderr, "%s: %s\n", title, message) }

func RevealFile(path string) error { return exec.Command("xdg-open", path).Start() }

func HideDockIcon() {}
