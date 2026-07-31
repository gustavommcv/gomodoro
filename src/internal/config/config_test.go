package config

import (
	"testing"
	"time"

	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

func TestDefaults(t *testing.T) {
	d := Defaults()

	t.Run("timer durations match the pomodoro package's canonical defaults", func(t *testing.T) {
		want := TimerConfig{
			Work:       time.Duration(pomodoro.DefaultWorkMinutes) * time.Minute,
			ShortBreak: time.Duration(pomodoro.DefaultShortBreakMinutes) * time.Minute,
			LongBreak:  time.Duration(pomodoro.DefaultLongBreakMinutes) * time.Minute,
		}
		if d.Timer != want {
			t.Errorf("got %+v, want %+v", d.Timer, want)
		}
	})

	t.Run("alerts default to both enabled", func(t *testing.T) {
		if !d.Alerts.Notify || !d.Alerts.Sound {
			t.Errorf("want both alerts enabled by default, got %+v", d.Alerts)
		}
	})

	t.Run("style defaults to the \"default\" theme", func(t *testing.T) {
		if d.Style.Theme != "default" {
			t.Errorf("got theme %q, want \"default\"", d.Style.Theme)
		}
	})

	t.Run("is stable across calls", func(t *testing.T) {
		if Defaults() != Defaults() {
			t.Error("Defaults() should be a pure, deterministic value")
		}
	})
}
