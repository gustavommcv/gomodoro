package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("a missing file at the default location is not an error -- defaults are used", func(t *testing.T) {
		dir := t.TempDir()
		loc := Location{Path: filepath.Join(dir, "config.lua"), Explicit: false}

		got, err := Load(loc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != Defaults() {
			t.Errorf("got %+v, want defaults %+v", got, Defaults())
		}
	})

	t.Run("a missing file at an explicit location is an error", func(t *testing.T) {
		dir := t.TempDir()
		loc := Location{Path: filepath.Join(dir, "config.lua"), Explicit: true}

		_, err := Load(loc)
		if err == nil {
			t.Fatal("expected an error for an explicitly-requested, missing config file")
		}
	})

	t.Run("an unreadable existing file is an error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.lua")
		if err := os.WriteFile(path, []byte("return {version = 1}"), 0o000); err != nil {
			t.Fatalf("setup: %v", err)
		}
		t.Cleanup(func() { os.Chmod(path, 0o644) })

		_, err := Load(Location{Path: path, Explicit: false})
		if err == nil {
			t.Skip("this platform does not enforce 0o000 permissions for the test's own user (common on Windows); skipping")
		}
	})

	t.Run("a present, valid file overlays the defaults", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{
			"config.lua": `return {version = 1, alerts = {sound = false}}`,
		})
		got, err := Load(Location{Path: filepath.Join(dir, "config.lua"), Explicit: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Alerts.Sound != false {
			t.Errorf("Alerts.Sound = %v, want false (from the file)", got.Alerts.Sound)
		}
		if got.Alerts.Notify != true {
			t.Errorf("Alerts.Notify = %v, want true (default, untouched)", got.Alerts.Notify)
		}
		if got.Timer != Defaults().Timer {
			t.Errorf("Timer = %+v, want unchanged defaults", got.Timer)
		}
	})

	t.Run("a syntactically invalid file is an error, even at the default location", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{"config.lua": "return {\n  version = ,\n}"})
		_, err := Load(Location{Path: filepath.Join(dir, "config.lua"), Explicit: false})
		if err == nil {
			t.Fatal("expected a syntax error")
		}
	})

	t.Run("a semantically invalid file is an error, even at the default location", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{"config.lua": `return {version = 1, timer = {work = "abc"}}`})
		_, err := Load(Location{Path: filepath.Join(dir, "config.lua"), Explicit: false})
		if err == nil {
			t.Fatal("expected a schema validation error")
		}
	})

	t.Run("modular config (settings/*.lua, themes/*.lua) loads correctly through Load", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{
			"config.lua": `
				local timer = require("settings.timer")
				return {version = 1, timer = timer}
			`,
			"settings/timer.lua": `return {work = "50m"}`,
		})
		got, err := Load(Location{Path: filepath.Join(dir, "config.lua"), Explicit: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Timer.Work != 50*time.Minute {
			t.Errorf("Timer.Work = %v, want 50m", got.Timer.Work)
		}
	})
}

func TestLoadIsNotExistErrorUnwraps(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(Location{Path: filepath.Join(dir, "nope.lua"), Explicit: true})
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want an error wrapping os.ErrNotExist, got: %v", err)
	}
}
