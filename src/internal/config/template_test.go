package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateDefaultFileMatchesDefaults is what keeps the generated
// template in sync with Defaults() over time: it doesn't compare source
// text, it round-trips the generated file through the exact same loader a
// real config file goes through and checks the result equals Defaults()
// exactly. If Defaults() ever changes without updating GenerateDefaultFile
// (or vice versa), this fails.
func TestGenerateDefaultFileMatchesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.lua")
	if err := os.WriteFile(path, GenerateDefaultFile(), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := Load(Location{Path: path, Explicit: true})
	if err != nil {
		t.Fatalf("the generated default file failed to load: %v", err)
	}
	if got != Defaults() {
		t.Errorf("loading the generated file gives %+v, want exactly Defaults() %+v", got, Defaults())
	}
}

func TestGenerateDefaultFileHasComments(t *testing.T) {
	content := string(GenerateDefaultFile())
	if !strings.Contains(content, "--") {
		t.Error("expected the generated file to include Lua comments")
	}
}

func TestWriteDefaultFile(t *testing.T) {
	t.Run("creates the directory and the file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nested", "gomodoro", "config.lua")

		if err := WriteDefaultFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected the file to exist: %v", err)
		}
	})

	t.Run("the written file loads to exactly Defaults()", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.lua")

		if err := WriteDefaultFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := Load(Location{Path: path, Explicit: true})
		if err != nil {
			t.Fatalf("unexpected error loading the written file: %v", err)
		}
		if got != Defaults() {
			t.Errorf("got %+v, want %+v", got, Defaults())
		}
	})

	t.Run("leaves no temporary file behind on success", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.lua")

		if err := WriteDefaultFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		if len(entries) != 1 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Errorf("expected exactly one file in the directory, got %v", names)
		}
	})
}
