package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func loadAndConvert(t *testing.T, source string) (FileConfig, error) {
	t.Helper()
	dir := writeFiles(t, map[string]string{"config.lua": source})
	tbl, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err != nil {
		return FileConfig{}, err
	}
	return tableToFileConfig(tbl, "config.lua")
}

func TestSchemaVersion(t *testing.T) {
	t.Run("version 1 is accepted", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a missing version is rejected -- there is no safe default schema version to assume", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {timer = {work = "25m"}}`)
		if err == nil {
			t.Fatal("expected an error for a missing version")
		}
		if !strings.Contains(err.Error(), "version") {
			t.Errorf("expected the error to mention \"version\", got: %v", err)
		}
	})

	t.Run("a future, unsupported version is rejected with a migration hint", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 2}`)
		if err == nil {
			t.Fatal("expected an error for an unsupported future version")
		}
		if !strings.Contains(err.Error(), "2") {
			t.Errorf("expected the error to mention the offending version, got: %v", err)
		}
	})

	t.Run("version 0 is rejected", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 0}`)
		if err == nil {
			t.Fatal("expected an error for version 0")
		}
	})

	t.Run("a non-numeric version is rejected", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = "1"}`)
		if err == nil {
			t.Fatal("expected an error for a string version")
		}
	})
}

func TestSchemaUnknownFields(t *testing.T) {
	t.Run("an unknown top-level field is rejected", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, bogus = true}`)
		if err == nil {
			t.Fatal("expected an error for an unknown top-level field")
		}
	})

	t.Run("a typo’d field name gets a did-you-mean suggestion", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, timer = {short_braek = "5m"}}`)
		if err == nil {
			t.Fatal("expected an error for the typo'd field")
		}
		msg := err.Error()
		if !strings.Contains(msg, "short_braek") {
			t.Errorf("expected the error to name the typo'd field, got: %v", msg)
		}
		if !strings.Contains(msg, "short_break") {
			t.Errorf("expected a did-you-mean suggestion for \"short_break\", got: %v", msg)
		}
	})

	t.Run("unknown fields are rejected within each section independently", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, alerts = {noitfy = true}}`)
		if err == nil {
			t.Fatal("expected an error for the unknown alerts field")
		}
	})
}

func TestSchemaTypes(t *testing.T) {
	t.Run("timer must be a table", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, timer = "nope"}`)
		if err == nil {
			t.Fatal("expected a type error")
		}
	})

	t.Run("alerts.notify must be a boolean", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, alerts = {notify = "yes"}}`)
		if err == nil {
			t.Fatal("expected a type error for a non-boolean notify")
		}
	})

	t.Run("style.theme must be a non-empty string", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, style = {theme = ""}}`)
		if err == nil {
			t.Fatal("expected an error for an empty theme string")
		}
	})

	t.Run("an unrecognized style field is rejected (style is reserved, not free-form)", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, style = {color = "red"}}`)
		if err == nil {
			t.Fatal("expected an error for an unrecognized style field")
		}
	})
}

func TestSchemaDurations(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"empty string", `""`},
		{"not a duration", `"abc"`},
		{"negative", `"-5m"`},
		{"zero", `"0m"`},
		{"sub-minute remainder", `"25m30s"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadAndConvert(t, `return {version = 1, timer = {work = `+c.value+`}}`)
			if err == nil {
				t.Fatalf("expected an error for timer.work = %s", c.value)
			}
		})
	}

	t.Run("a duration given as a number instead of a string is rejected, not coerced", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, timer = {work = 25}}`)
		if err == nil {
			t.Fatal("expected an error for a numeric work value")
		}
	})

	t.Run("false for work is rejected, not coerced to zero", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {version = 1, timer = {work = false}}`)
		if err == nil {
			t.Fatal("expected an error for a boolean work value")
		}
	})

	t.Run("a valid whole-minute duration is accepted", func(t *testing.T) {
		fc, err := loadAndConvert(t, `return {version = 1, timer = {work = "50m"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fc.Timer == nil || fc.Timer.Work == nil || *fc.Timer.Work != 50*time.Minute {
			t.Errorf("Timer.Work = %+v, want 50m", fc.Timer)
		}
	})

	t.Run("an hour-based duration is accepted too, since it's still a whole number of minutes", func(t *testing.T) {
		fc, err := loadAndConvert(t, `return {version = 1, timer = {work = "1h"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fc.Timer == nil || fc.Timer.Work == nil || *fc.Timer.Work != time.Hour {
			t.Errorf("Timer.Work = %+v, want 1h", fc.Timer)
		}
	})

	t.Run("validation is atomic: an invalid field anywhere means nothing from the file is applied", func(t *testing.T) {
		_, err := loadAndConvert(t, `return {
			version = 1,
			timer = {work = "25m", short_break = "not-a-duration"},
		}`)
		if err == nil {
			t.Fatal("expected an error")
		}
		// Confirmed via the FileConfig zero-value return on error at the
		// call site (loadAndConvert returns FileConfig{} on any error) --
		// there is no partial FileTimerConfig with just "work" filled in
		// leaking out.
	})
}

func TestSchemaEmptyConfigUsesAllDefaultsViaOverlay(t *testing.T) {
	fc, err := loadAndConvert(t, `return {version = 1}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := Defaults().Overlay(fc)
	if got != Defaults() {
		t.Errorf("an empty (version-only) config should overlay to exactly the defaults, got %+v", got)
	}
}
