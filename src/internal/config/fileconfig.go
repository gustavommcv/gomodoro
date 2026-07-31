package config

import "time"

// FileConfig is what a config file can express: an overlay on top of
// Defaults(). Every field is a pointer (down to the section level and the
// individual field level) so "not present in the file" (nil) can never be
// confused with a present-but-zero-ish value (empty string, false, zero
// duration) -- Go's zero value is never used as a stand-in for absence.
// This presence-tracking is intentionally contained to this one type: the
// rest of the domain only ever sees a fully-resolved Config.
type FileConfig struct {
	Version int
	Timer   *FileTimerConfig
	Alerts  *FileAlertConfig
	Style   *FileStyleConfig
}

type FileTimerConfig struct {
	Work       *time.Duration
	ShortBreak *time.Duration
	LongBreak  *time.Duration
}

type FileAlertConfig struct {
	Notify *bool
	Sound  *bool
}

type FileStyleConfig struct {
	Theme *string
}

// Overlay applies f on top of c: a present section's present fields
// replace c's; anything absent (a nil section, or a nil field within a
// present section) leaves c's existing value untouched.
func (c Config) Overlay(f FileConfig) Config {
	if f.Timer != nil {
		if f.Timer.Work != nil {
			c.Timer.Work = *f.Timer.Work
		}
		if f.Timer.ShortBreak != nil {
			c.Timer.ShortBreak = *f.Timer.ShortBreak
		}
		if f.Timer.LongBreak != nil {
			c.Timer.LongBreak = *f.Timer.LongBreak
		}
	}

	if f.Alerts != nil {
		if f.Alerts.Notify != nil {
			c.Alerts.Notify = *f.Alerts.Notify
		}
		if f.Alerts.Sound != nil {
			c.Alerts.Sound = *f.Alerts.Sound
		}
	}

	if f.Style != nil {
		if f.Style.Theme != nil {
			c.Style.Theme = *f.Style.Theme
		}
	}

	return c
}
