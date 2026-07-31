package config

import (
	"path/filepath"
	"testing"
)

func TestValidateModuleName(t *testing.T) {
	valid := []string{
		"settings",
		"settings.timer",
		"settings.alerts",
		"themes.default",
		"a.b.c.d",
		"_private.module",
		"module_2",
	}
	for _, name := range valid {
		t.Run("valid: "+name, func(t *testing.T) {
			if err := validateModuleName(name); err != nil {
				t.Errorf("validateModuleName(%q) = %v, want nil", name, err)
			}
		})
	}

	invalid := []string{
		"",
		".",
		"..",
		"settings.",
		".settings",
		"settings..timer",
		"settings/timer",  // forward slash: ambiguous separator, dots only
		"settings\\timer", // backslash: same
		"/settings",       // absolute path
		"../secrets",      // parent traversal
		"settings...timer",
		"1settings", // must not start with a digit
		"settings.1timer",
		"set tings", // space
		"settings.timer ",
		" settings",
		"settings.timer\x00", // NUL byte
	}
	for _, name := range invalid {
		t.Run("invalid: "+name, func(t *testing.T) {
			if err := validateModuleName(name); err == nil {
				t.Errorf("validateModuleName(%q) = nil, want an error", name)
			}
		})
	}
}

func TestModuleNameToRelPath(t *testing.T) {
	cases := map[string]string{
		"settings":       "settings.lua",
		"settings.timer": filepath.Join("settings", "timer.lua"),
		"a.b.c":          filepath.Join("a", "b", "c.lua"),
	}
	for name, want := range cases {
		if got := moduleNameToRelPath(name); got != want {
			t.Errorf("moduleNameToRelPath(%q) = %q, want %q", name, got, want)
		}
	}
}
