package pomodoro

import "time"

// phaseResult distinguishes how a phase's countdown ended, so a caller can
// tell a normal completion apart from an explicit skip or a cancellation.
// Only a completion should trigger the end-of-phase alerts; a skip gets a
// short text acknowledgement instead, and a cancellation gets neither.
type phaseResult int

const (
	phaseCompleted phaseResult = iota
	phaseSkipped
	phaseCancelled
)

// phaseState is the running/paused state of a phase's countdown while it
// is active. It is distinct from phaseResult: Running and Paused are
// transient states a countdown passes through, while Completed/Skipped/
// Cancelled are the final outcomes once it stops.
type phaseState int

const (
	phaseRunning phaseState = iota
	phasePaused
)

// interactiveCommand is a single decoded keyboard command recognized while
// a phase's countdown is running with interactive controls.
type interactiveCommand int

const (
	cmdNone interactiveCommand = iota
	cmdTogglePause
	cmdSkip
	cmdQuit
)

// translateKey maps a single raw input byte to the command it represents,
// or cmdNone if the byte isn't one of Gomodoro's three recognized controls.
// It deliberately does not attempt to decode multi-byte escape sequences
// (arrow keys, function keys, etc.) -- Gomodoro only needs single printable
// letters and Ctrl+C, not general terminal input parsing.
func translateKey(b byte) interactiveCommand {
	switch b {
	case 'p', 'P':
		return cmdTogglePause
	case 's', 'S':
		return cmdSkip
	case 0x03: // ETX -- Ctrl+C as a literal byte. In raw mode the terminal
		// no longer turns this into SIGINT on its own (ISIG is off), so
		// the countdown has to recognize it itself.
		return cmdQuit
	default:
		return cmdNone
	}
}

// phaseTimer tracks a phase's remaining time against a fixed deadline. It
// has no notion of "paused" itself -- shift is the only way its notion of
// elapsed time changes, and it is the caller's job (applyTogglePause) to
// call shift with exactly the duration spent paused when resuming. This
// keeps the time arithmetic isolated from the running/paused state machine
// and from any concurrency, so it can be tested as plain, deterministic
// function calls.
type phaseTimer struct {
	start    time.Time
	deadline time.Time
}

func newPhaseTimer(start time.Time, total time.Duration) *phaseTimer {
	return &phaseTimer{start: start, deadline: start.Add(total)}
}

// remaining returns the time left until the deadline as of now, clamped to
// zero once the deadline has passed.
func (t *phaseTimer) remaining(now time.Time) time.Duration {
	remaining := t.deadline.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// shift moves both the start and the deadline forward by d. Shifting both
// by the same amount preserves the phase's total duration while re-basing
// tick scheduling (which is anchored to start) to account for time spent
// paused -- this is what avoids a burst of catch-up ticks immediately
// after a resume.
func (t *phaseTimer) shift(d time.Duration) {
	t.start = t.start.Add(d)
	t.deadline = t.deadline.Add(d)
}

// applyTogglePause toggles between Running and Paused. Pausing records the
// current time as pausedAt; resuming shifts timer forward by exactly the
// time spent paused, so remaining() reads the same value immediately after
// resuming that it did the moment before pausing -- no drift, regardless
// of how long the pause lasted.
func applyTogglePause(state phaseState, pausedAt time.Time, now time.Time, timer *phaseTimer) (phaseState, time.Time) {
	if state == phaseRunning {
		return phasePaused, now
	}
	timer.shift(now.Sub(pausedAt))
	return phaseRunning, time.Time{}
}
