//go:build darwin

package ui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void micoff_hide_dock(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[[NSApplication sharedApplication] setActivationPolicy:NSApplicationActivationPolicyAccessory];
	});
}
*/
import "C"

import (
	"os/exec"
	"strings"
)

// Alert shows a modal error dialog.
func Alert(title, message string) {
	q := func(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }
	script := `display alert "` + q(title) + `" message "` + q(message) + `" as warning`
	_ = exec.Command("osascript", "-e", script).Run()
}

// RevealFile shows the file in Finder.
func RevealFile(path string) error {
	return exec.Command("open", "-R", path).Run()
}

// HideDockIcon makes the app menu-bar-only (also set via LSUIElement in the
// bundle; this covers running the bare binary during development).
func HideDockIcon() { C.micoff_hide_dock() }
