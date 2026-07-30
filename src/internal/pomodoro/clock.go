package pomodoro

import "time"

// Clock abstracts the passage of time so a phase's progress loop can be
// driven instantly and deterministically in tests, instead of waiting on
// the real clock.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
