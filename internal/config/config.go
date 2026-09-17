// Package config persists the user's settings as a small JSON file in the
// OS-standard per-user config directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultHotkey = "ctrl+alt+m"

type Config struct {
	Hotkey       string `json:"hotkey"`
	StartAtLogin bool   `json:"start_at_login"`
}

func Default() Config {
	return Config{Hotkey: DefaultHotkey}
}

// Path returns the config file location, e.g.
//
//	macOS:   ~/Library/Application Support/mic-off/config.json
//	Windows: %APPDATA%\mic-off\config.json
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mic-off", "config.json"), nil
}

// Load reads the config at path. A missing file yields the defaults; a
// corrupt file yields the defaults plus an error so the caller can warn.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Hotkey == "" {
		cfg.Hotkey = DefaultHotkey
	}
	return cfg, nil
}

// Save writes the config atomically, creating the directory if needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
