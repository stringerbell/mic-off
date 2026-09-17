//go:build windows

package mic

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	ole "github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// coreAudioWin mutes the default capture endpoints through the Windows Core
// Audio API. This is the same switch as the mute button in Sound settings.
type coreAudioWin struct {
	mu sync.Mutex
}

// New returns the Windows implementation.
func New() (Mic, error) {
	m := &coreAudioWin{}
	if _, err := m.Muted(); err != nil {
		return nil, err
	}
	return m, nil
}

func comInit() (func(), error) {
	runtime.LockOSThread()
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		var oe *ole.OleError
		// S_FALSE (already initialized) and RPC_E_CHANGED_MODE are fine.
		if !errors.As(err, &oe) || (oe.Code() != 1 && oe.Code() != 0x80010106) {
			runtime.UnlockOSThread()
			return nil, err
		}
	}
	return func() {
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}, nil
}

// endpoints yields the endpoint-volume interface of every distinct default
// capture device (the "communications" default that Zoom/Teams use, and the
// general default), so both end up muted.
func endpoints(fn func(*wca.IAudioEndpointVolume) error) error {
	done, err := comInit()
	if err != nil {
		return err
	}
	defer done()

	var enum *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &enum); err != nil {
		return fmt.Errorf("audio device enumerator: %w", err)
	}
	defer enum.Release()

	seen := map[string]bool{}
	found := 0
	for _, role := range []uint32{wca.ECommunications, wca.EConsole} {
		var dev *wca.IMMDevice
		if err := enum.GetDefaultAudioEndpoint(wca.ECapture, role, &dev); err != nil {
			continue // no default device for this role
		}
		var id string
		dev.GetId(&id)
		if seen[id] {
			dev.Release()
			continue
		}
		seen[id] = true
		var vol *wca.IAudioEndpointVolume
		err := dev.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &vol)
		dev.Release()
		if err != nil {
			return fmt.Errorf("endpoint volume: %w", err)
		}
		err = fn(vol)
		vol.Release()
		if err != nil {
			return err
		}
		found++
	}
	if found == 0 {
		return errors.New("no default microphone found")
	}
	return nil
}

func (m *coreAudioWin) Muted() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	first := true
	muted := false
	err := endpoints(func(v *wca.IAudioEndpointVolume) error {
		if !first {
			return nil // state follows the communications device
		}
		first = false
		return v.GetMute(&muted)
	})
	return muted, err
}

func (m *coreAudioWin) SetMuted(muted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return endpoints(func(v *wca.IAudioEndpointVolume) error {
		return v.SetMute(muted, nil)
	})
}

func (m *coreAudioWin) Description() string { return "default microphone (Windows endpoint mute)" }
