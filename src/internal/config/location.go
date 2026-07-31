package config

import (
	"fmt"
	"path/filepath"
)

// ConfigFileName is the name of Gomodoro's main configuration file, always
// looked for directly inside the config directory (never searched for in
// the current working directory -- see Location.Explicit and ResolveLocation).
const ConfigFileName = "config.lua"

// EnvVar is the environment variable that selects the config file, with
// lower precedence than --config but higher than the default location.
const EnvVar = "GOMODORO_CONFIG"

// Location is a resolved config file path plus whether it was explicitly
// requested (via --config or GOMODORO_CONFIG) or is just the computed
// default location. The distinction matters for error handling: a missing
// file is fine at the default location (fall back to defaults silently)
// but an error everywhere else (the user asked for a specific file).
type Location struct {
	Path     string
	Explicit bool
}

// ResolveLocation applies Gomodoro's config file precedence: an explicit
// --config flag value, then the GOMODORO_CONFIG environment variable, then
// the platform's standard per-user config directory (via userConfigDir,
// i.e. os.UserConfigDir in real use) joined with "gomodoro/config.lua".
// getenv and userConfigDir are injected so this is testable without
// touching the real environment or filesystem.
func ResolveLocation(flagValue string, getenv func(string) string, userConfigDir func() (string, error)) (Location, error) {
	if flagValue != "" {
		return Location{Path: flagValue, Explicit: true}, nil
	}

	if env := getenv(EnvVar); env != "" {
		return Location{Path: env, Explicit: true}, nil
	}

	dir, err := userConfigDir()
	if err != nil {
		return Location{}, fmt.Errorf("determine default config directory: %w", err)
	}
	return Location{
		Path:     filepath.Join(dir, "gomodoro", ConfigFileName),
		Explicit: false,
	}, nil
}
