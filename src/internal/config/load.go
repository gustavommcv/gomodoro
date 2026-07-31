package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Load resolves Gomodoro's effective configuration at loc: if loc.Path
// doesn't exist and loc wasn't explicitly requested (the default
// location), that's not an error -- Defaults() is returned as-is. An
// explicitly-requested file that's missing, unreadable, or invalid
// (syntax or schema) is always an error; the caller must not start the
// timer, alerts, or raw mode in that case.
//
// This is the only place the rest of Gomodoro needs to call: Lua, paths,
// and precedence are all resolved here, and the return value is a plain,
// validated Config.
func Load(loc Location) (Config, error) {
	if _, err := os.Stat(loc.Path); err != nil {
		if os.IsNotExist(err) && !loc.Explicit {
			return Defaults(), nil
		}
		return Config{}, fmt.Errorf("open config file %s: %w", loc.Path, err)
	}

	tbl, err := loadLuaConfig(loc.Path)
	if err != nil {
		return Config{}, err
	}

	fc, err := tableToFileConfig(tbl, filepath.Base(loc.Path))
	if err != nil {
		return Config{}, err
	}

	return Defaults().Overlay(fc), nil
}
