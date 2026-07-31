// Package config resolves Gomodoro's effective configuration: compiled-in
// defaults, optionally overlaid by a Lua configuration file, optionally
// overlaid again by explicit CLI flags. Everything downstream of Load
// receives a plain, validated Config value -- nothing outside this package
// needs to know about Lua, file paths, or precedence rules.
package config

import (
	"time"

	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

// Config is Gomodoro's fully-resolved, typed configuration.
type Config struct {
	Timer  TimerConfig
	Alerts AlertConfig
	Style  StyleConfig
}

// TimerConfig holds the three phase durations.
type TimerConfig struct {
	Work       time.Duration
	ShortBreak time.Duration
	LongBreak  time.Duration
}

// AlertConfig holds the two independent end-of-phase alert toggles.
type AlertConfig struct {
	Notify bool
	Sound  bool
}

// StyleConfig is reserved for a future visual theming feature. Today it
// carries a theme name but has no visual effect; "default" is the only
// value that currently exists.
type StyleConfig struct {
	Theme string
}

// Defaults returns Gomodoro's compiled-in default configuration. This is
// the single canonical source: the CLI's flag defaults, the config loader's
// overlay base, and "gomodoro config init"'s generated file all derive from
// this value rather than duplicating it.
func Defaults() Config {
	return Config{
		Timer: TimerConfig{
			Work:       time.Duration(pomodoro.DefaultWorkMinutes) * time.Minute,
			ShortBreak: time.Duration(pomodoro.DefaultShortBreakMinutes) * time.Minute,
			LongBreak:  time.Duration(pomodoro.DefaultLongBreakMinutes) * time.Minute,
		},
		Alerts: AlertConfig{
			Notify: true,
			Sound:  true,
		},
		Style: StyleConfig{
			Theme: "default",
		},
	}
}
