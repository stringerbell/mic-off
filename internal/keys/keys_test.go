package keys

import "testing"

func TestParseCanonical(t *testing.T) {
	s, err := Parse("ctrl+alt+m")
	if err != nil {
		t.Fatal(err)
	}
	want := Spec{Ctrl: true, Alt: true, Key: "m"}
	if s != want {
		t.Fatalf("got %+v want %+v", s, want)
	}
	if s.String() != "ctrl+alt+m" {
		t.Fatalf("String() = %q", s.String())
	}
}

func TestParseAliasesCaseAndSpaces(t *testing.T) {
	s, err := Parse(" Command + Option + Shift + F5 ")
	if err != nil {
		t.Fatal(err)
	}
	want := Spec{Cmd: true, Alt: true, Shift: true, Key: "f5"}
	if s != want {
		t.Fatalf("got %+v want %+v", s, want)
	}
	// Win/super map to cmd so a config file written on one OS reads on the other.
	s, _ = Parse("win+space")
	if !s.Cmd || s.Key != "space" {
		t.Fatalf("got %+v", s)
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	for _, bad := range []string{"", "m", "5", "ctrl+", "ctrl+alt", "bogus+m", "ctrl+f99", "ctrl+enter"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}

func TestFunctionKeysNeedNoModifier(t *testing.T) {
	s, err := Parse("f13")
	if err == nil {
		t.Fatal("f13 is outside the supported F1-F12 range")
	}
	s, err = Parse("f12")
	if err != nil {
		t.Fatal(err)
	}
	if s.HasModifier() {
		t.Fatal("unexpected modifier")
	}
}

func TestWithModifierAndKeyAreCopies(t *testing.T) {
	base := Spec{Ctrl: true, Key: "m"}
	changed := base.WithModifier(Shift, true).WithKey("k")
	if base.Shift || base.Key != "m" {
		t.Fatal("base mutated")
	}
	if !changed.Shift || !changed.Ctrl || changed.Key != "k" {
		t.Fatalf("got %+v", changed)
	}
	if changed.WithModifier(Ctrl, false).Ctrl {
		t.Fatal("could not clear modifier")
	}
}

func TestDisplay(t *testing.T) {
	s := Spec{Ctrl: true, Alt: true, Key: "m"}
	if got := Display(s, "darwin"); got != "⌃⌥M" {
		t.Errorf("darwin: %q", got)
	}
	if got := Display(s, "windows"); got != "Ctrl+Alt+M" {
		t.Errorf("windows: %q", got)
	}
	if got := Display(Spec{Cmd: true, Key: "space"}, "windows"); got != "Win+Space" {
		t.Errorf("windows space: %q", got)
	}
	if got := Display(Spec{Key: "f12"}, "darwin"); got != "F12" {
		t.Errorf("darwin f12: %q", got)
	}
}

// The recorder shows the modifiers being held before a key is pressed, so a
// Spec without a Key must render cleanly (no trailing "+").
func TestDisplayModifiersOnly(t *testing.T) {
	held := Spec{Ctrl: true, Alt: true}
	if got := Display(held, "darwin"); got != "⌃⌥" {
		t.Errorf("darwin: %q", got)
	}
	if got := Display(held, "windows"); got != "Ctrl+Alt" {
		t.Errorf("windows: %q", got)
	}
	if got := Display(Spec{}, "windows"); got != "" {
		t.Errorf("empty: %q", got)
	}
}

func TestKeyListIsCompleteAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Keys {
		if seen[k] {
			t.Fatalf("duplicate key %q", k)
		}
		seen[k] = true
		if err := (Spec{Ctrl: true, Key: k}).Validate(); err != nil {
			t.Errorf("key %q from Keys does not validate: %v", k, err)
		}
	}
	if len(Keys) != 26+10+12+1 {
		t.Fatalf("got %d keys", len(Keys))
	}
}
