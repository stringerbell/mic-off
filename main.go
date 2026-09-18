// mic-off mutes your microphone at the operating-system level with a global
// hotkey, so meeting apps that only "soft mute" get real silence.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/systray"

	"micoff/internal/app"
	"micoff/internal/autostart"
	"micoff/internal/config"
	"micoff/internal/hotkey"
	"micoff/internal/icon"
	"micoff/internal/keys"
	"micoff/internal/mic"
	"micoff/internal/recorder"
	"micoff/internal/ui"
)

var version = "dev"

func main() {
	toggle := flag.Bool("toggle", false, "toggle the microphone and exit")
	mute := flag.Bool("mute", false, "mute the microphone and exit")
	unmute := flag.Bool("unmute", false, "unmute the microphone and exit")
	status := flag.Bool("status", false, "print \"muted\" or \"live\" and exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("mic-off", version)
		return
	}
	if *toggle || *mute || *unmute || *status {
		os.Exit(runCLI(*toggle, *mute, *unmute))
	}
	runTray()
}

// runCLI is the scriptable one-shot mode (Stream Deck, shell aliases, ...).
func runCLI(toggle, mute, unmute bool) int {
	m, err := mic.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mic-off:", err)
		return 1
	}
	cur, err := m.Muted()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mic-off:", err)
		return 1
	}
	want := cur
	switch {
	case toggle:
		want = !cur
	case mute:
		want = true
	case unmute:
		want = false
	}
	if want != cur {
		if err := m.SetMuted(want); err != nil {
			fmt.Fprintln(os.Stderr, "mic-off:", err)
			return 1
		}
	}
	fmt.Println(stateWord(want))
	return 0
}

func stateWord(muted bool) string {
	if muted {
		return "muted"
	}
	return "live"
}

func runTray() {
	cfgPath, err := config.Path()
	if err != nil {
		ui.Alert("mic-off", "Cannot find a settings folder: "+err.Error())
		os.Exit(1)
	}
	setupLogging(filepath.Join(filepath.Dir(cfgPath), "mic-off.log"))
	log.Printf("mic-off %s starting on %s/%s", version, runtime.GOOS, runtime.GOARCH)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("settings file problem, using defaults: %v", err)
	}
	m, err := mic.New()
	if err != nil {
		log.Printf("microphone: %v", err)
		ui.Alert("mic-off", "Cannot control the microphone: "+err.Error())
		os.Exit(1)
	}
	log.Printf("controlling %s", m.Description())

	var t *tray
	ctrl := app.New(app.Deps{
		Mic:       m,
		Hotkeys:   hotkey.New(),
		Autostart: autostart.Autostart{},
		Config:    cfg,
		Save:      func(c config.Config) error { return config.Save(cfgPath, c) },
		Logf:      log.Printf,
		OnState:   func(muted bool) { t.setState(muted) },
		OnHotkey:  func(spec keys.Spec, err error) { t.setHotkey(spec, err) },
	})

	systray.Run(func() {
		ui.HideDockIcon()
		t = newTray(ctrl, cfgPath)
		ctrl.Start()
		go func() {
			for range time.Tick(time.Second) {
				ctrl.Refresh()
			}
		}()
	}, func() {
		ctrl.Stop()
		log.Println("mic-off exiting")
	})
}

func setupLogging(path string) {
	os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}

// tray is the menu bar / notification area UI.
type tray struct {
	ctrl   *app.Controller
	icons  icon.Set
	status *systray.MenuItem
	toggle *systray.MenuItem
	hotkey *systray.MenuItem
	login  *systray.MenuItem
}

func newTray(ctrl *app.Controller, cfgPath string) *tray {
	t := &tray{
		ctrl:  ctrl,
		icons: icon.ForPlatform(runtime.GOOS),
	}
	t.setState(false)

	t.status = systray.AddMenuItem("Microphone: live", "")
	t.status.Disable()
	t.toggle = systray.AddMenuItem("Mute microphone", "Same as pressing the hotkey")
	onClick(t.toggle, func() {
		if _, err := ctrl.Toggle(); err != nil {
			ui.Alert("mic-off", err.Error())
		}
	})

	systray.AddSeparator()
	t.hotkey = systray.AddMenuItem("Change hotkey…", "Press the key combination you want to use")
	onClick(t.hotkey, t.changeHotkey)

	systray.AddSeparator()
	t.login = systray.AddMenuItemCheckbox("Start at login", "", ctrl.Config().StartAtLogin)
	onClick(t.login, func() {
		on := !t.login.Checked()
		if err := ctrl.SetStartAtLogin(on); err != nil {
			ui.Alert("mic-off", "Could not change login setting: "+err.Error())
			return
		}
		setChecked(t.login, on)
	})
	settings := systray.AddMenuItem("Show settings file", cfgPath)
	onClick(settings, func() {
		if _, err := os.Stat(cfgPath); err != nil {
			config.Save(cfgPath, ctrl.Config())
		}
		ui.RevealFile(cfgPath)
	})

	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit mic-off", "")
	onClick(quit, systray.Quit)
	return t
}

func onClick(item *systray.MenuItem, fn func()) {
	go func() {
		for range item.ClickedCh {
			fn()
		}
	}()
}

func setChecked(item *systray.MenuItem, on bool) {
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}

// setState updates the icon and the status/toggle rows.
func (t *tray) setState(muted bool) {
	combo := keys.Display(t.ctrl.Spec(), runtime.GOOS)
	if muted {
		systray.SetIcon(t.icons.Muted)
		systray.SetTooltip("mic-off: microphone is MUTED (" + combo + " to unmute)")
	} else if t.icons.LiveIsTemplate {
		systray.SetTemplateIcon(t.icons.Live, t.icons.Live)
		systray.SetTooltip("mic-off: microphone is live (" + combo + " to mute)")
	} else {
		systray.SetIcon(t.icons.Live)
		systray.SetTooltip("mic-off: microphone is live (" + combo + " to mute)")
	}
	if t.status == nil {
		return
	}
	if muted {
		t.status.SetTitle("Microphone: MUTED")
		t.toggle.SetTitle("Unmute microphone")
	} else {
		t.status.SetTitle("Microphone: live")
		t.toggle.SetTitle("Mute microphone")
	}
}

func (t *tray) setHotkey(keys.Spec, error) { t.render() }

// changeHotkey opens the capture window. The current combo is released
// while it is open so pressing it there is captured rather than toggling
// the mic; Resume puts it back unless the user saved a new one.
func (t *tray) changeHotkey() {
	t.ctrl.Suspend()
	defer t.ctrl.Resume()
	if err := recorder.Show(t.ctrl.Spec(), t.ctrl.SetSpec); err != nil {
		ui.Alert("mic-off", "Cannot open the hotkey window: "+err.Error())
	}
}

// render syncs the menu rows with the controller's state.
func (t *tray) render() {
	if t.hotkey == nil {
		return
	}
	combo := keys.Display(t.ctrl.Spec(), runtime.GOOS)
	if t.ctrl.HotkeyError() != nil {
		t.hotkey.SetTitle("Change hotkey (" + combo + " isn't working)…")
	} else {
		t.hotkey.SetTitle("Change hotkey (" + combo + ")…")
	}
	t.setState(t.ctrl.Muted())
}
