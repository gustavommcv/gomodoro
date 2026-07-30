package pomodoro

import (
	"testing"
	"time"
)

func TestTranslateKey(t *testing.T) {
	cases := []struct {
		name string
		key  byte
		want interactiveCommand
	}{
		{"lowercase p toggles pause", 'p', cmdTogglePause},
		{"uppercase P toggles pause", 'P', cmdTogglePause},
		{"lowercase s skips", 's', cmdSkip},
		{"uppercase S skips", 'S', cmdSkip},
		{"Ctrl+C (ETX, 0x03) quits", 0x03, cmdQuit},
		{"an unrecognized key is ignored", 'x', cmdNone},
		{"a digit is ignored", '5', cmdNone},
		{"a newline is ignored (that's WaitForUser's key, not a command)", '\n', cmdNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := translateKey(c.key); got != c.want {
				t.Errorf("translateKey(%q) = %v, want %v", c.key, got, c.want)
			}
		})
	}
}

func TestPhaseTimer(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("remaining counts down linearly while running", func(t *testing.T) {
		timer := newPhaseTimer(base, 10*time.Minute)

		if got := timer.remaining(base); got != 10*time.Minute {
			t.Errorf("remaining at start = %v, want 10m", got)
		}
		if got := timer.remaining(base.Add(4 * time.Minute)); got != 6*time.Minute {
			t.Errorf("remaining after 4m = %v, want 6m", got)
		}
		if got := timer.remaining(base.Add(10 * time.Minute)); got != 0 {
			t.Errorf("remaining at deadline = %v, want 0", got)
		}
	})

	t.Run("remaining never goes negative past the deadline", func(t *testing.T) {
		timer := newPhaseTimer(base, 1*time.Minute)
		if got := timer.remaining(base.Add(5 * time.Minute)); got != 0 {
			t.Errorf("remaining well past deadline = %v, want 0 (clamped)", got)
		}
	})

	t.Run("shifting by a pause duration preserves remaining exactly", func(t *testing.T) {
		timer := newPhaseTimer(base, 25*time.Minute)
		atPause := base.Add(10 * time.Minute)
		remainingAtPause := timer.remaining(atPause)

		pauseLength := 3 * time.Minute
		timer.shift(pauseLength)

		// The same wall-clock instant now reports the same remaining time
		// it did the moment before the shift...
		if got := timer.remaining(atPause); got != remainingAtPause+pauseLength {
			t.Errorf("remaining at the old pause instant after shift = %v, want %v (shift moved the deadline forward)", got, remainingAtPause+pauseLength)
		}
		// ...and at "pause instant + pause length" (i.e. the moment resume
		// actually happens), remaining is exactly what it was at pause time.
		if got := timer.remaining(atPause.Add(pauseLength)); got != remainingAtPause {
			t.Errorf("remaining at resume time = %v, want %v (preserved across the pause)", got, remainingAtPause)
		}
	})

	t.Run("a pause longer than the phase's original duration introduces no drift", func(t *testing.T) {
		timer := newPhaseTimer(base, 5*time.Minute)
		atPause := base.Add(1 * time.Minute)
		remainingAtPause := timer.remaining(atPause) // 4m

		hugePause := 2 * time.Hour
		timer.shift(hugePause)

		if got := timer.remaining(atPause.Add(hugePause)); got != remainingAtPause {
			t.Errorf("remaining after a huge pause = %v, want %v (unchanged)", got, remainingAtPause)
		}
	})

	t.Run("many short pause/resume cycles introduce no cumulative drift", func(t *testing.T) {
		timer := newPhaseTimer(base, 25*time.Minute)
		now := base.Add(5 * time.Minute)
		remainingBefore := timer.remaining(now)

		for i := 0; i < 20; i++ {
			pause := 37 * time.Second // an arbitrary, non-round duration
			timer.shift(pause)
			now = now.Add(pause)
		}

		if got := timer.remaining(now); got != remainingBefore {
			t.Errorf("remaining after 20 pause/resume cycles = %v, want %v (no drift)", got, remainingBefore)
		}
	})
}

func TestApplyTogglePause(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("pausing a running phase records the pause instant without touching the timer", func(t *testing.T) {
		// phaseTimer has no notion of "paused" itself -- freezing what's
		// displayed while paused is the caller's job (it simply stops
		// asking the timer for a fresh `remaining` until resume). What
		// applyTogglePause must get right here is: entering Paused does
		// NOT call shift -- only resuming does -- so the timer itself is
		// provably untouched by the act of pausing.
		timer := newPhaseTimer(base, 10*time.Minute)
		deadlineBeforePause := timer.deadline
		pausedAt := base.Add(2 * time.Minute)

		state, at := applyTogglePause(phaseRunning, time.Time{}, pausedAt, timer)

		if state != phasePaused {
			t.Fatalf("state = %v, want phasePaused", state)
		}
		if !at.Equal(pausedAt) {
			t.Errorf("pausedAt = %v, want %v", at, pausedAt)
		}
		if !timer.deadline.Equal(deadlineBeforePause) {
			t.Errorf("deadline changed on pause (%v -> %v); only resuming should shift it", deadlineBeforePause, timer.deadline)
		}
	})

	t.Run("resuming a paused phase preserves remaining time exactly", func(t *testing.T) {
		timer := newPhaseTimer(base, 10*time.Minute)
		pausedAt := base.Add(2 * time.Minute)
		remainingAtPause := timer.remaining(pausedAt)

		resumeAt := pausedAt.Add(90 * time.Second)
		state, _ := applyTogglePause(phasePaused, pausedAt, resumeAt, timer)

		if state != phaseRunning {
			t.Fatalf("state = %v, want phaseRunning", state)
		}
		if got := timer.remaining(resumeAt); got != remainingAtPause {
			t.Errorf("remaining at resume = %v, want %v (preserved)", got, remainingAtPause)
		}
	})

	t.Run("multiple pause/resume cycles preserve remaining time each time", func(t *testing.T) {
		timer := newPhaseTimer(base, 10*time.Minute)
		state := phaseRunning
		var pausedAt time.Time
		var frozenRemaining time.Duration
		now := base

		checkpoints := []time.Duration{1 * time.Minute, 30 * time.Second, 45 * time.Second, 2 * time.Minute}
		for i, gap := range checkpoints {
			now = now.Add(gap)

			if state == phaseRunning {
				// About to pause: remember what "remaining" reads right
				// now, since it must read exactly the same once resumed,
				// no matter how long the pause lasts.
				frozenRemaining = timer.remaining(now)
			}

			state, pausedAt = applyTogglePause(state, pausedAt, now, timer)

			if i%2 == 0 {
				if state != phasePaused {
					t.Fatalf("cycle %d: state = %v, want phasePaused", i, state)
				}
			} else {
				if state != phaseRunning {
					t.Fatalf("cycle %d: state = %v, want phaseRunning", i, state)
				}
				if got := timer.remaining(now); got != frozenRemaining {
					t.Errorf("cycle %d: remaining after resume = %v, want %v", i, got, frozenRemaining)
				}
			}
		}
	})
}
