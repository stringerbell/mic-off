package mic

import "sync"

// Fake is an in-memory Mic for tests.
type Fake struct {
	mu    sync.Mutex
	muted bool
	// Err, when set, is returned by every call.
	Err   error
	Calls []bool
}

func (f *Fake) Muted() (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.muted, f.Err
}

func (f *Fake) SetMuted(m bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return f.Err
	}
	f.muted = m
	f.Calls = append(f.Calls, m)
	return nil
}

// SetExternally simulates another program changing the mute state.
func (f *Fake) SetExternally(m bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.muted = m
}

func (f *Fake) Description() string { return "fake microphone" }
