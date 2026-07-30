// Package pomodoro implements the Gomodoro cycle rules: a lap of four work
// phases, with a short break after each of the first three and a long break
// after the fourth.
package pomodoro

import (
	"context"
	"fmt"
)

const (
	DefaultWorkMinutes       = 25
	DefaultShortBreakMinutes = 5
	DefaultLongBreakMinutes  = 15
	CyclesPerLap             = 4
)

// Sleeper runs a single Pomodoro phase, or waits for the user to confirm
// moving on to the next one. DefaultSleeper is the real, terminal-facing
// implementation; tests use a smaller double to verify cycle ordering
// without touching the terminal, the clock, or the notification system.
type Sleeper interface {
	Work(ctx context.Context, minutes int) error
	ShortBreak(ctx context.Context, minutes int) error
	LongBreak(ctx context.Context, minutes int) error
	WaitForUser(ctx context.Context) error
}

// Pomodoro runs one lap: four work phases, a short break after each of the
// first three, and a long break after the fourth. It returns as soon as a
// phase returns an error, including context cancellation.
func Pomodoro(ctx context.Context, sleeper Sleeper, work, shortBreak, longBreak int) error {
	if work <= 0 {
		return fmt.Errorf("work duration must be greater than zero, got %d", work)
	}
	if shortBreak <= 0 {
		return fmt.Errorf("short break duration must be greater than zero, got %d", shortBreak)
	}
	if longBreak <= 0 {
		return fmt.Errorf("long break duration must be greater than zero, got %d", longBreak)
	}

	for cycle := 1; cycle <= CyclesPerLap; cycle++ {
		if err := sleeper.Work(ctx, work); err != nil {
			return err
		}

		if cycle == CyclesPerLap {
			if err := sleeper.LongBreak(ctx, longBreak); err != nil {
				return err
			}
		} else {
			if err := sleeper.ShortBreak(ctx, shortBreak); err != nil {
				return err
			}
		}
	}

	return nil
}
