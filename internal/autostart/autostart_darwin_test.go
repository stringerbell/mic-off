//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableWritesAndRemovesLaunchAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "Library", "LaunchAgents", label+".plist")

	if err := (Autostart{}).Enable(true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := executable()
	body := string(data)
	for _, want := range []string{"<key>RunAtLoad</key><true/>", "<string>" + exe + "</string>", "<string>" + label + "</string>"} {
		if !strings.Contains(body, want) {
			t.Errorf("plist missing %q:\n%s", want, body)
		}
	}

	if err := (Autostart{}).Enable(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("plist should be removed")
	}
	// Disabling when already disabled is not an error.
	if err := (Autostart{}).Enable(false); err != nil {
		t.Fatal(err)
	}
}
