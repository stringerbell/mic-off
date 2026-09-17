// Package mic mutes the default microphone at the operating-system level, so
// every application receives silence regardless of its own mute button.
package mic

// Mic controls the OS default input device.
type Mic interface {
	// Muted reports whether the default input is currently muted.
	Muted() (bool, error)
	// SetMuted mutes or unmutes the default input.
	SetMuted(muted bool) error
	// Description names the device/mechanism for display in logs.
	Description() string
}
