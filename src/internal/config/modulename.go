package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// moduleNamePattern is the complete allow-list for a require() module
// name: dot-separated segments, each starting with a letter or underscore
// and containing only letters, digits, or underscores. This single
// pattern -- rather than a set of ad-hoc rejections -- is what proves no
// path traversal, absolute path, empty segment, or ambiguous separator
// (/, \) can ever be encoded in a name that passes validation: none of
// those characters appear in the pattern at all.
var moduleNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

// validateModuleName rejects anything that isn't a plain, dotted,
// alphanumeric module name.
func validateModuleName(name string) error {
	if !moduleNamePattern.MatchString(name) {
		return fmt.Errorf("invalid module name %q: must be dot-separated letters, digits, and underscores only (e.g. \"settings.timer\")", name)
	}
	return nil
}

// moduleNameToRelPath converts a validated module name to the relative
// file path it resolves to, e.g. "settings.timer" -> "settings/timer.lua"
// (using the OS's own separator, never a hardcoded one).
func moduleNameToRelPath(name string) string {
	segments := strings.Split(name, ".")
	segments[len(segments)-1] += ".lua"
	return filepath.Join(segments...)
}
