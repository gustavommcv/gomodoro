package pomodoro

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gustavommcv/gomodoro/src/pkg/beep"
	"github.com/gustavommcv/gomodoro/src/pkg/progressbar"
	"github.com/gustavommcv/gomodoro/src/pkg/sound"
)

// DefaultSleeper is the real, terminal-facing Sleeper: it renders a
// progress bar roughly once per second, alerts the user at the end of a
// phase (desktop notification and/or a short sound, each independently
// toggleable), and waits for the user to press Enter before returning.
type DefaultSleeper struct {
	ProgressWriter io.Writer
	ErrWriter      io.Writer
	InputReader    io.Reader

	// Clock controls timing; if nil, the real clock is used. Tests inject a
	// fake clock so phases complete instantly instead of waiting for real time.
	Clock Clock

	// Notify sends a desktop notification; if nil, notifications are skipped.
	// Tests inject a stub so no real OS notification is fired.
	Notify func(title, message string) error

	// PlaySound plays a short alert sound; if nil, the sound is skipped.
	// Tests inject a stub so no real OS sound is fired. It takes ctx so
	// playback stops promptly on Ctrl+C instead of blocking the phase from
	// ending.
	PlaySound func(ctx context.Context) error

	// NotifyEnabled and SoundEnabled are independent: disabling one must
	// not disable the other.
	NotifyEnabled bool
	SoundEnabled  bool

	// Interactive enables raw-mode keyboard controls (pause/resume/skip)
	// during a phase's countdown. It should only be true when stdin and
	// stdout are both real terminals -- the caller decides that, not this
	// type.
	Interactive bool

	// EnterRawMode puts the terminal into raw/uncooked input mode and
	// returns a function that restores it; nil (or a non-nil error from a
	// call) means raw mode is unavailable, in which case the countdown
	// falls back to the plain, non-interactive display. Tests inject a
	// no-op stub so no real terminal is touched.
	EnterRawMode func() (restore func(), err error)

	// inputOnce/inputCh back the single shared input pump (see input()).
	inputOnce sync.Once
	inputCh   chan byte
}

// NewDefaultSleeper wires a DefaultSleeper for real use: the real clock,
// real desktop notifications and alert sounds (gated by notifyEnabled and
// soundEnabled respectively), and interactive raw-mode controls gated by
// interactive.
func NewDefaultSleeper(progress, errOut io.Writer, input io.Reader, notifyEnabled, soundEnabled, interactive bool, enterRawMode func() (func(), error)) *DefaultSleeper {
	return &DefaultSleeper{
		ProgressWriter: progress,
		ErrWriter:      errOut,
		InputReader:    input,
		Clock:          realClock{},
		Notify:         beep.SendNotification,
		PlaySound:      sound.Play,
		NotifyEnabled:  notifyEnabled,
		SoundEnabled:   soundEnabled,
		Interactive:    interactive,
		EnterRawMode:   enterRawMode,
	}
}

// input lazily starts the single background goroutine that reads raw bytes
// from InputReader for the lifetime of this DefaultSleeper, and returns the
// channel it publishes to. There is exactly one such goroutine per
// DefaultSleeper: WaitForUser and the interactive countdown both read from
// this same channel, at different (non-overlapping) points in a phase's
// lifecycle, which is what avoids two goroutines racing to read the same
// stdin. Reading from InputReader cannot itself be interrupted mid-syscall,
// so -- like the rest of Gomodoro's input handling -- this goroutine may
// still be blocked in a Read call after nothing is consuming its output
// anymore (e.g. after the process starts exiting); it is not forcibly
// stopped, only left to end on its own via EOF/error, or with the process.
func (d *DefaultSleeper) input() <-chan byte {
	d.inputOnce.Do(func() {
		ch := make(chan byte, 32)
		d.inputCh = ch
		go func() {
			defer close(ch)
			buf := make([]byte, 1)
			for {
				n, err := d.InputReader.Read(buf)
				if n > 0 {
					ch <- buf[0]
				}
				if err != nil {
					return
				}
			}
		}()
	})
	return d.inputCh
}

// drain discards any bytes already buffered on ch without blocking. It is
// used when a phase's interactive countdown starts listening, so a
// keypress left over from before it started (e.g. typed during the
// previous phase's "Press Enter" wait) is never misread as a command for a
// phase it wasn't intended for.
func drain(ch <-chan byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func (d *DefaultSleeper) clock() Clock {
	if d.Clock == nil {
		return realClock{}
	}
	return d.Clock
}

func (d *DefaultSleeper) errWriter() io.Writer {
	if d.ErrWriter == nil {
		return io.Discard
	}
	return d.ErrWriter
}

func (d *DefaultSleeper) Work(ctx context.Context, minutes int) error {
	return d.runPhase(ctx, minutes, "Work", "Work Finished")
}

func (d *DefaultSleeper) ShortBreak(ctx context.Context, minutes int) error {
	return d.runPhase(ctx, minutes, "Short Break", "Short break Finished")
}

func (d *DefaultSleeper) LongBreak(ctx context.Context, minutes int) error {
	return d.runPhase(ctx, minutes, "Long Break", "Long break Finished")
}

func (d *DefaultSleeper) runPhase(ctx context.Context, minutes int, label, notifyTitle string) error {
	fmt.Fprintf(d.ProgressWriter, "\n%s for %d minutes\n", label, minutes)

	result, err := d.countdown(ctx, minutes, label)
	if err != nil {
		return err
	}

	// A skipped phase gets a short acknowledgement instead of the
	// completion alerts, and moves straight on -- the user just made an
	// explicit, deliberate keypress, unlike a natural completion where
	// they may not be at the keyboard, so re-confirming with Enter would
	// just be a redundant extra step here.
	if result == phaseSkipped {
		return nil
	}

	d.sendAlerts(ctx, notifyTitle)

	return d.WaitForUser(ctx)
}

// countdown runs a phase's timer, with interactive pause/resume/skip
// controls when d.Interactive is true and raw mode is actually available;
// otherwise it falls back to the plain, non-interactive countdown
// unchanged from before this feature existed.
func (d *DefaultSleeper) countdown(ctx context.Context, minutes int, label string) (phaseResult, error) {
	if !d.Interactive || d.EnterRawMode == nil {
		if err := d.showProgress(ctx, minutes, label); err != nil {
			return phaseCancelled, err
		}
		return phaseCompleted, nil
	}

	restore, err := d.EnterRawMode()
	if err != nil {
		// Raw mode was expected to work but didn't (an unusual terminal,
		// most likely) -- degrade to the non-interactive countdown rather
		// than failing the whole phase over a control-scheme nicety.
		if err := d.showProgress(ctx, minutes, label); err != nil {
			return phaseCancelled, err
		}
		return phaseCompleted, nil
	}
	defer restore()

	input := d.input()
	drain(input)

	return d.runInteractiveCountdown(ctx, minutes, label, input)
}

// sendAlerts fires the notification and sound independently: either can
// fail, or be disabled, without affecting the other. Both are best-effort —
// a failure is a warning on stderr, never a reason to stop the Pomodoro. A
// cancelled ctx is not treated as a failure (no warning): WaitForUser right
// after this will itself observe the same cancellation and return promptly.
func (d *DefaultSleeper) sendAlerts(ctx context.Context, notifyTitle string) {
	if d.NotifyEnabled && d.Notify != nil {
		if err := d.Notify(notifyTitle, "press enter in the terminal to continue"); err != nil {
			fmt.Fprintf(d.errWriter(), "gomodoro: warning: %v\n", err)
		}
	}

	if d.SoundEnabled && d.PlaySound != nil {
		if err := d.PlaySound(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(d.errWriter(), "gomodoro: warning: %v\n", err)
		}
	}
}

// WaitForUser blocks until the user presses Enter, or ctx is cancelled,
// whichever happens first. It reads from the same shared input pump the
// interactive countdown uses (see input()), so there is never more than
// one goroutine reading InputReader at a time even across phases.
func (d *DefaultSleeper) WaitForUser(ctx context.Context) error {
	fmt.Fprint(d.ProgressWriter, "\nPress Enter to continue...")

	input := d.input()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case b, ok := <-input:
			if !ok || b == '\n' {
				return nil
			}
		}
	}
}

// showProgress renders a progress bar once per simulated second until
// minutes have elapsed, or ctx is cancelled. Ticks are anchored to a fixed
// deadline (start + N seconds) rather than chained one-second sleeps, so
// rendering time does not accumulate into clock drift over a long phase.
func (d *DefaultSleeper) showProgress(ctx context.Context, totalMinutes int, label string) error {
	clock := d.clock()
	total := time.Duration(totalMinutes) * time.Minute
	start := clock.Now()
	deadline := start.Add(total)

	tick := 0
	for {
		remaining := deadline.Sub(clock.Now())
		if remaining < 0 {
			remaining = 0
		}

		fmt.Fprint(d.ProgressWriter, "\r")
		progressbar.ProgressBar(d.ProgressWriter, float64(total-remaining), float64(total))
		fmt.Fprintf(d.ProgressWriter, " - %s: %02d:%02d remaining",
			label, int(remaining/time.Minute), int(remaining/time.Second)%60)

		if remaining <= 0 {
			break
		}

		tick++
		next := start.Add(time.Duration(tick) * time.Second)
		wait := next.Sub(clock.Now())
		if wait < 0 {
			wait = 0
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-clock.After(wait):
		}
	}

	fmt.Fprintln(d.ProgressWriter)
	return nil
}

// runInteractiveCountdown is showProgress's pause/resume/skip-aware
// counterpart. It is parameterized over its input channel (rather than
// calling d.input() itself) so it can be driven deterministically in tests
// without a real terminal, real goroutine timing, or a real clock.
//
// Time bookkeeping is delegated entirely to phaseTimer/applyTogglePause
// (see interactive.go), which are plain, synchronous, and independently
// tested; this function's own job is just the concurrency glue: render,
// wait for whichever happens first (a tick, a command, or cancellation),
// and react.
func (d *DefaultSleeper) runInteractiveCountdown(ctx context.Context, totalMinutes int, label string, input <-chan byte) (phaseResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clk := d.clock()
	timer := newPhaseTimer(clk.Now(), time.Duration(totalMinutes)*time.Minute)

	state := phaseRunning
	var pausedAt time.Time

	tick := 0
	for {
		remaining := timer.remaining(clk.Now())
		d.renderInteractive(remaining, time.Duration(totalMinutes)*time.Minute, label, state)

		if remaining <= 0 {
			fmt.Fprintln(d.ProgressWriter)
			return phaseCompleted, nil
		}

		var waitCh <-chan time.Time
		if state == phaseRunning {
			tick++
			next := timer.start.Add(time.Duration(tick) * time.Second)
			wait := next.Sub(clk.Now())
			if wait < 0 {
				wait = 0
			}
			waitCh = clk.After(wait)
		}

		select {
		case <-ctx.Done():
			fmt.Fprintln(d.ProgressWriter)
			return phaseCancelled, ctx.Err()

		case b, ok := <-input:
			if !ok {
				input = nil // stdin closed: stop selecting on it, keep ticking
				continue
			}
			switch translateKey(b) {
			case cmdSkip:
				fmt.Fprintf(d.ProgressWriter, "\n%s skipped.\n", label)
				return phaseSkipped, nil
			case cmdQuit:
				cancel()
				fmt.Fprintln(d.ProgressWriter)
				return phaseCancelled, ctx.Err()
			case cmdTogglePause:
				state, pausedAt = applyTogglePause(state, pausedAt, clk.Now(), timer)
			}

		case <-waitCh:
		}
	}
}

// renderInteractive draws the single status line shown during an
// interactive countdown: the same percent/bar/label/remaining format
// showProgress uses, plus the available controls, redrawn via a leading
// \r exactly like the rest of Gomodoro's progress display (no other ANSI
// is used). Trailing padding covers the case where this line is shorter
// than the previous one it's overwriting (e.g. leaving the longer "PAUSED
// ... Resume" text when resuming) so no fragment of it is left visible.
func (d *DefaultSleeper) renderInteractive(remaining, total time.Duration, label string, state phaseState) {
	fmt.Fprint(d.ProgressWriter, "\r")
	progressbar.ProgressBar(d.ProgressWriter, float64(total-remaining), float64(total))
	fmt.Fprintf(d.ProgressWriter, " - %s: %02d:%02d remaining",
		label, int(remaining/time.Minute), int(remaining/time.Second)%60)

	if state == phasePaused {
		fmt.Fprint(d.ProgressWriter, "  PAUSED  [p] Resume  [s] Skip  [Ctrl+C] Quit          ")
	} else {
		fmt.Fprint(d.ProgressWriter, "  [p] Pause  [s] Skip  [Ctrl+C] Quit          ")
	}
}
