//go:build darwin

package mic

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <string.h>

static AudioObjectPropertyAddress mo_addr(UInt32 sel, UInt32 scope, UInt32 el) {
	AudioObjectPropertyAddress a = { sel, scope, el };
	return a;
}

static OSStatus mo_default_input(AudioDeviceID *out) {
	AudioObjectPropertyAddress a = mo_addr(kAudioHardwarePropertyDefaultInputDevice,
		kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	UInt32 sz = sizeof(*out);
	return AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &sz, out);
}

// mo_settable reports whether an input-scope property exists and can be set.
static int mo_settable(AudioDeviceID dev, UInt32 sel, UInt32 el) {
	AudioObjectPropertyAddress a = mo_addr(sel, kAudioObjectPropertyScopeInput, el);
	if (!AudioObjectHasProperty(dev, &a)) return 0;
	Boolean s = 0;
	if (AudioObjectIsPropertySettable(dev, &a, &s) != noErr) return 0;
	return s ? 1 : 0;
}

static OSStatus mo_get_u32(AudioDeviceID dev, UInt32 sel, UInt32 el, UInt32 *out) {
	AudioObjectPropertyAddress a = mo_addr(sel, kAudioObjectPropertyScopeInput, el);
	UInt32 sz = sizeof(*out);
	return AudioObjectGetPropertyData(dev, &a, 0, NULL, &sz, out);
}

static OSStatus mo_set_u32(AudioDeviceID dev, UInt32 sel, UInt32 el, UInt32 v) {
	AudioObjectPropertyAddress a = mo_addr(sel, kAudioObjectPropertyScopeInput, el);
	return AudioObjectSetPropertyData(dev, &a, 0, NULL, sizeof(v), &v);
}

static OSStatus mo_get_f32(AudioDeviceID dev, UInt32 sel, UInt32 el, Float32 *out) {
	AudioObjectPropertyAddress a = mo_addr(sel, kAudioObjectPropertyScopeInput, el);
	UInt32 sz = sizeof(*out);
	return AudioObjectGetPropertyData(dev, &a, 0, NULL, &sz, out);
}

static OSStatus mo_set_f32(AudioDeviceID dev, UInt32 sel, UInt32 el, Float32 v) {
	AudioObjectPropertyAddress a = mo_addr(sel, kAudioObjectPropertyScopeInput, el);
	return AudioObjectSetPropertyData(dev, &a, 0, NULL, sizeof(v), &v);
}

static void mo_name(AudioDeviceID dev, char *buf, int len) {
	buf[0] = 0;
	AudioObjectPropertyAddress a = mo_addr(kAudioObjectPropertyName,
		kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	CFStringRef s = NULL;
	UInt32 sz = sizeof(s);
	if (AudioObjectGetPropertyData(dev, &a, 0, NULL, &sz, &s) != noErr || s == NULL) return;
	CFStringGetCString(s, buf, len, kCFStringEncodingUTF8);
	CFRelease(s);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
)

// coreAudio drives the default input device. It prefers the device's real
// mute switch (kAudioDevicePropertyMute) and falls back to driving the input
// level to zero for devices that have no mute control.
type coreAudio struct {
	mu          sync.Mutex
	savedVolume float32 // level to restore after a volume-based mute
}

// New returns the macOS implementation.
func New() (Mic, error) {
	m := &coreAudio{savedVolume: 0.75}
	if _, err := m.device(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *coreAudio) device() (C.AudioDeviceID, error) {
	var dev C.AudioDeviceID
	if st := C.mo_default_input(&dev); st != 0 || dev == 0 {
		return 0, fmt.Errorf("no default input device (CoreAudio status %d)", int(st))
	}
	return dev, nil
}

// elements returns the input elements (0 = main, then per channel) on which
// the property can be set. If the main element supports it, only it is used.
func elements(dev C.AudioDeviceID, sel C.UInt32) []C.UInt32 {
	if C.mo_settable(dev, sel, 0) != 0 {
		return []C.UInt32{0}
	}
	var els []C.UInt32
	for el := C.UInt32(1); el <= 2; el++ {
		if C.mo_settable(dev, sel, el) != 0 {
			els = append(els, el)
		}
	}
	return els
}

func (m *coreAudio) Muted() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev, err := m.device()
	if err != nil {
		return false, err
	}
	if els := elements(dev, C.kAudioDevicePropertyMute); len(els) > 0 {
		for _, el := range els {
			var v C.UInt32
			if st := C.mo_get_u32(dev, C.kAudioDevicePropertyMute, el, &v); st != 0 {
				return false, fmt.Errorf("reading mute (status %d)", int(st))
			}
			if v == 0 {
				return false, nil
			}
		}
		return true, nil
	}
	els := elements(dev, C.kAudioDevicePropertyVolumeScalar)
	if len(els) == 0 {
		return false, errors.New("the default input device has neither a mute nor a volume control")
	}
	for _, el := range els {
		var v C.Float32
		if st := C.mo_get_f32(dev, C.kAudioDevicePropertyVolumeScalar, el, &v); st != 0 {
			return false, fmt.Errorf("reading input level (status %d)", int(st))
		}
		if v > 0.001 {
			return false, nil
		}
	}
	return true, nil
}

func (m *coreAudio) SetMuted(muted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev, err := m.device()
	if err != nil {
		return err
	}
	if els := elements(dev, C.kAudioDevicePropertyMute); len(els) > 0 {
		var v C.UInt32
		if muted {
			v = 1
		}
		for _, el := range els {
			if st := C.mo_set_u32(dev, C.kAudioDevicePropertyMute, el, v); st != 0 {
				return fmt.Errorf("setting mute (status %d)", int(st))
			}
		}
		return nil
	}
	els := elements(dev, C.kAudioDevicePropertyVolumeScalar)
	if len(els) == 0 {
		return errors.New("the default input device has neither a mute nor a volume control")
	}
	if muted {
		// Remember the loudest channel so unmute restores what the user had.
		var maxV C.Float32
		for _, el := range els {
			var v C.Float32
			if C.mo_get_f32(dev, C.kAudioDevicePropertyVolumeScalar, el, &v) == 0 && v > maxV {
				maxV = v
			}
		}
		if maxV > 0.001 {
			m.savedVolume = float32(maxV)
		}
	}
	level := C.Float32(0)
	if !muted {
		level = C.Float32(m.savedVolume)
	}
	for _, el := range els {
		if st := C.mo_set_f32(dev, C.kAudioDevicePropertyVolumeScalar, el, level); st != 0 {
			return fmt.Errorf("setting input level (status %d)", int(st))
		}
	}
	return nil
}

func (m *coreAudio) Description() string {
	dev, err := m.device()
	if err != nil {
		return err.Error()
	}
	var buf [256]C.char
	C.mo_name(dev, &buf[0], C.int(len(buf)))
	name := C.GoString(&buf[0])
	how := "input level"
	if len(elements(dev, C.kAudioDevicePropertyMute)) > 0 {
		how = "hardware mute"
	}
	return fmt.Sprintf("%s (%s)", name, how)
}
