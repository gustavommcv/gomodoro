package config

import (
	"testing"
	"time"
)

func TestFileConfigOverlay(t *testing.T) {
	t.Run("an empty FileConfig changes nothing", func(t *testing.T) {
		got := Defaults().Overlay(FileConfig{})
		if got != Defaults() {
			t.Errorf("got %+v, want unchanged defaults %+v", got, Defaults())
		}
	})

	t.Run("a present section overlays only its own fields, others in the same section keep defaults", func(t *testing.T) {
		work := 50 * time.Minute
		got := Defaults().Overlay(FileConfig{
			Timer: &FileTimerConfig{Work: &work},
		})

		if got.Timer.Work != work {
			t.Errorf("Timer.Work = %v, want %v", got.Timer.Work, work)
		}
		if got.Timer.ShortBreak != Defaults().Timer.ShortBreak {
			t.Errorf("Timer.ShortBreak = %v, want default %v (untouched)", got.Timer.ShortBreak, Defaults().Timer.ShortBreak)
		}
		if got.Timer.LongBreak != Defaults().Timer.LongBreak {
			t.Errorf("Timer.LongBreak = %v, want default %v (untouched)", got.Timer.LongBreak, Defaults().Timer.LongBreak)
		}
	})

	t.Run("an absent section is left entirely at defaults", func(t *testing.T) {
		got := Defaults().Overlay(FileConfig{Alerts: nil})
		if got.Alerts != Defaults().Alerts {
			t.Errorf("Alerts = %+v, want unchanged defaults %+v", got.Alerts, Defaults().Alerts)
		}
	})

	t.Run("explicit false is applied, not confused with absence", func(t *testing.T) {
		f := false
		got := Defaults().Overlay(FileConfig{Alerts: &FileAlertConfig{Sound: &f}})

		if got.Alerts.Sound != false {
			t.Errorf("Alerts.Sound = %v, want false", got.Alerts.Sound)
		}
		if got.Alerts.Notify != true {
			t.Errorf("Alerts.Notify = %v, want true (untouched default, not flipped by the other field's presence)", got.Alerts.Notify)
		}
	})

	t.Run("explicit true is applied even though it matches the default (not treated as absence)", func(t *testing.T) {
		trueVal := true
		got := Defaults().Overlay(FileConfig{Alerts: &FileAlertConfig{Notify: &trueVal}})
		if got.Alerts.Notify != true {
			t.Errorf("Alerts.Notify = %v, want true", got.Alerts.Notify)
		}
	})

	t.Run("style theme overlays independently", func(t *testing.T) {
		theme := "custom"
		got := Defaults().Overlay(FileConfig{Style: &FileStyleConfig{Theme: &theme}})
		if got.Style.Theme != "custom" {
			t.Errorf("Style.Theme = %q, want \"custom\"", got.Style.Theme)
		}
	})

	t.Run("all three sections overlay together without interfering", func(t *testing.T) {
		work := 50 * time.Minute
		sound := false
		theme := "custom"
		got := Defaults().Overlay(FileConfig{
			Timer:  &FileTimerConfig{Work: &work},
			Alerts: &FileAlertConfig{Sound: &sound},
			Style:  &FileStyleConfig{Theme: &theme},
		})

		want := Config{
			Timer:  TimerConfig{Work: work, ShortBreak: Defaults().Timer.ShortBreak, LongBreak: Defaults().Timer.LongBreak},
			Alerts: AlertConfig{Notify: true, Sound: false},
			Style:  StyleConfig{Theme: "custom"},
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}
