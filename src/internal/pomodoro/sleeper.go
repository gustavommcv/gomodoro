package pomodoro

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gustavommcv/gomodoro/src/pkg/beep"
	"github.com/gustavommcv/gomodoro/src/pkg/progressbar"
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
	// Tests inject a stub so no real OS sound is fired.
	PlaySound func() error

	// NotifyEnabled and SoundEnabled are independent: disabling one must
	// not disable the other.
	NotifyEnabled bool
	SoundEnabled  bool
}

// NewDefaultSleeper wires a DefaultSleeper for real use: the real clock and
// real desktop notifications and alert sounds, gated by notifyEnabled and
// soundEnabled respectively.
func NewDefaultSleeper(progress, errOut io.Writer, input io.Reader, notifyEnabled, soundEnabled bool) *DefaultSleeper {
	return &DefaultSleeper{
		ProgressWriter: progress,
		ErrWriter:      errOut,
		InputReader:    input,
		Clock:          realClock{},
		Notify:         beep.SendNotification,
		PlaySound:      beep.PlaySound,
		NotifyEnabled:  notifyEnabled,
		SoundEnabled:   soundEnabled,
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

	if err := d.showProgress(ctx, minutes, label); err != nil {
		return err
	}

	d.sendAlerts(notifyTitle)

	return d.WaitForUser(ctx)
}

// sendAlerts fires the notification and sound independently: either can
// fail, or be disabled, without affecting the other. Both are best-effort —
// a failure is a warning on stderr, never a reason to stop the Pomodoro.
func (d *DefaultSleeper) sendAlerts(notifyTitle string) {
	if d.NotifyEnabled && d.Notify != nil {
		if err := d.Notify(notifyTitle, "press enter in the terminal to continue"); err != nil {
			fmt.Fprintf(d.errWriter(), "gomodoro: warning: %v\n", err)
		}
	}

	if d.SoundEnabled && d.PlaySound != nil {
		if err := d.PlaySound(); err != nil {
			fmt.Fprintf(d.errWriter(), "gomodoro: warning: %v\n", err)
		}
	}
}

// WaitForUser blocks until the user presses Enter, or ctx is cancelled,
// whichever happens first. Reading from InputReader cannot itself be
// interrupted mid-syscall, so the read runs in a short-lived goroutine that
// this call does not wait for once ctx is cancelled; it exits on its own as
// soon as the reader produces data, an error, or is closed by the caller.
func (d *DefaultSleeper) WaitForUser(ctx context.Context) error {
	fmt.Fprint(d.ProgressWriter, "\nPress Enter to continue...")

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1)
		for {
			n, err := d.InputReader.Read(buf)
			if err != nil || (n > 0 && buf[0] == '\n') {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
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
