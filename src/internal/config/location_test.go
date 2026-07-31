package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func fakeUserConfigDir(dir string, err error) func() (string, error) {
	return func() (string, error) { return dir, err }
}

func fakeGetenv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestResolveLocation(t *testing.T) {
	t.Run("no flag, no env: resolves to the default location and is not explicit", func(t *testing.T) {
		loc, err := ResolveLocation("", fakeGetenv(nil), fakeUserConfigDir("/home/user/.config", nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join("/home/user/.config", "gomodoro", "config.lua")
		if loc.Path != want {
			t.Errorf("Path = %q, want %q", loc.Path, want)
		}
		if loc.Explicit {
			t.Error("want Explicit = false for the default location")
		}
	})

	t.Run("--config takes precedence over everything else", func(t *testing.T) {
		loc, err := ResolveLocation(
			"/explicit/config.lua",
			fakeGetenv(map[string]string{"GOMODORO_CONFIG": "/env/config.lua"}),
			fakeUserConfigDir("/home/user/.config", nil),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loc.Path != "/explicit/config.lua" {
			t.Errorf("Path = %q, want the --config value", loc.Path)
		}
		if !loc.Explicit {
			t.Error("want Explicit = true for an explicitly-provided path")
		}
	})

	t.Run("GOMODORO_CONFIG is used when --config is absent", func(t *testing.T) {
		loc, err := ResolveLocation(
			"",
			fakeGetenv(map[string]string{"GOMODORO_CONFIG": "/env/config.lua"}),
			fakeUserConfigDir("/home/user/.config", nil),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loc.Path != "/env/config.lua" {
			t.Errorf("Path = %q, want the GOMODORO_CONFIG value", loc.Path)
		}
		if !loc.Explicit {
			t.Error("want Explicit = true for GOMODORO_CONFIG too (it's an explicit selection, just via env instead of a flag)")
		}
	})

	t.Run("default location uses os.UserConfigDir()'s answer, not a hardcoded path", func(t *testing.T) {
		loc, err := ResolveLocation("", fakeGetenv(nil), fakeUserConfigDir("C:\\Users\\someone\\AppData\\Roaming", nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join("C:\\Users\\someone\\AppData\\Roaming", "gomodoro", "config.lua")
		if loc.Path != want {
			t.Errorf("Path = %q, want %q", loc.Path, want)
		}
	})

	t.Run("a failure to determine the default location surfaces as an error", func(t *testing.T) {
		wantErr := errors.New("no config dir on this platform")
		_, err := ResolveLocation("", fakeGetenv(nil), fakeUserConfigDir("", wantErr))
		if !errors.Is(err, wantErr) {
			t.Fatalf("want error wrapping %v, got %v", wantErr, err)
		}
	})
}
