//go:build !darwin && !windows

package mic

import "errors"

// New reports that this platform is not supported yet.
func New() (Mic, error) {
	return nil, errors.New("mic-off supports macOS and Windows only")
}
