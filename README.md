# mic-off

Zoom, Meet and Teams don't actually turn your microphone off when you click
mute. They keep listening (that's how the "You're muted, are you talking?"
popup works). **mic-off** mutes the microphone at the operating-system level,
below every app, with one key combination. Muted means silence, everywhere.

One small file, nothing to install, no accounts, no permissions prompts.

## Use it

### macOS

1. Download `mic-off-macos.zip` and double-click it to unpack `mic-off.app`.
2. Drag `mic-off.app` into your **Applications** folder.
3. **First launch only:** right-click `mic-off.app` and choose **Open**, then
   **Open** again. (On macOS 15 or newer: double-click it, dismiss the warning,
   open **System Settings → Privacy & Security**, scroll down and click
   **Open Anyway**.) macOS shows this warning for any app that isn't from
   the App Store; it only happens once.
4. A microphone icon appears in the menu bar. Press **Control + Option + M**
   to mute; press it again to unmute. The icon turns red with a slash while
   muted.

### Windows

1. Download `mic-off-windows-x64.exe` (or `-arm64` on an ARM laptop) and put
   it anywhere, for example your Documents folder.
2. Double-click it. If Windows shows "Windows protected your PC", click
   **More info** then **Run anyway**. That happens once.
3. A microphone icon appears in the notification area (bottom right; click
   the `^` arrow if it's hidden). Press **Ctrl + Alt + M** to mute; press it
   again to unmute.

### Changing the hotkey

Click the microphone icon → **Hotkey**. Tick the modifiers you want and pick
the key under **Key**. It takes effect immediately and is remembered.
Function keys (F1–F12) can be used on their own; anything else needs at
least one modifier so ordinary typing isn't hijacked.

### Other menu items

- **Mute / Unmute microphone**: same as the hotkey.
- **Start at login**: launch mic-off automatically.
- **Show settings file**: opens the folder holding `config.json` and
  `mic-off.log`, handy if something goes wrong.

## How it mutes

- **macOS:** uses the input device's own mute switch through CoreAudio (the
  same control as the mute button in System Settings → Sound → Input). A few
  microphones have no mute switch; for those mic-off drops the input level to
  zero and restores the previous level on unmute. If your mic is one of
  those, turn off **"Automatically adjust microphone volume"** in Zoom so it
  doesn't push the level back up.
- **Windows:** sets the mute flag on the default recording device(s) through
  Windows Core Audio, the same as muting in Sound settings. Both the
  "default" and "default communications" devices are muted if they differ.

The icon polls the real device state every second, so if you unmute from
System Settings or another app, the icon follows.

## Command line

The same file is scriptable, which is useful with a Stream Deck or a shell alias:

```
mic-off -toggle     # flip and print the new state
mic-off -mute
mic-off -unmute
mic-off -status     # prints "muted" or "live"
```

On macOS the binary is inside the bundle: `/Applications/mic-off.app/Contents/MacOS/mic-off`.

## Building from source

Requires Go 1.22+ and, on macOS, Xcode command line tools.

```
make test-unit     # unit tests
make vet           # type-check macOS and Windows targets
make run           # run the tray app from source
make build         # dist/mic-off.app, dist/mic-off-macos.zip and the Windows .exe files
```

Windows binaries are cross-compiled from macOS with no C toolchain. Pushing
a tag like `v0.2.0` builds and attaches all artifacts to a GitHub release
(see `.github/workflows/release.yml`).

### Signing

The macOS build is ad-hoc signed, which is why the first-launch warning
appears. To remove it, set `CODESIGN_IDENTITY` to a Developer ID
Application certificate when running `make build-mac` and notarize the zip
with `xcrun notarytool`.
