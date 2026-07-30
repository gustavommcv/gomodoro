package pomodoro

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// fakeClock advances instantly: After(d) reports the elapsed duration
// immediately instead of waiting on a real timer, so timing logic can be
// exercised without slowing down the test suite.
type fakeClock struct {
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.now = c.now.Add(d)
	ch := make(chan time.Time, 1)
	ch <- c.now
	return ch
}

func newTestSleeper(clock Clock) (*DefaultSleeper, *bytes.Buffer, *bytes.Buffer) {
	var progress, errs bytes.Buffer
	sleeper := &DefaultSleeper{
		ProgressWriter: &progress,
		ErrWriter:      &errs,
		InputReader:    strings.NewReader("\n"),
		Clock:          clock,
		Notify:         func(title, message string) error { return nil },
		PlaySound:      func(ctx context.Context) error { return nil },
		NotifyEnabled:  true,
		SoundEnabled:   true,
	}
	return sleeper, &progress, &errs
}

func TestDefaultSleeperShowProgress(t *testing.T) {
	t.Run("completes instantly with a fake clock and reaches 100%", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, progress, _ := newTestSleeper(clock)

		start := time.Now()
		err := sleeper.showProgress(context.Background(), 5, "Work")
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if elapsed > time.Second {
			t.Fatalf("showProgress used the real clock, took %v", elapsed)
		}
		if !strings.Contains(progress.String(), "100%") {
			t.Errorf("expected progress output to reach 100%%, got: %q", progress.String())
		}
		if !strings.Contains(progress.String(), "00:00 remaining") {
			t.Errorf("expected progress output to end at 00:00 remaining, got: %q", progress.String())
		}
	})

	t.Run("ticks roughly once per simulated second without drift", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, _, _ := newTestSleeper(clock)

		if err := sleeper.showProgress(context.Background(), 1, "Work"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantEnd := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
		if !clock.Now().Equal(wantEnd) {
			t.Errorf("want simulated clock to land exactly on %v, got %v", wantEnd, clock.Now())
		}
	})

	t.Run("stops immediately when the context is cancelled", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, _, _ := newTestSleeper(clock)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := sleeper.showProgress(ctx, 25, "Work")

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	})
}

func TestDefaultSleeperWaitForUser(t *testing.T) {
	t.Run("returns once Enter is pressed", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		sleeper.InputReader = strings.NewReader("x\n")

		if err := sleeper.WaitForUser(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns immediately when the context is cancelled, without blocking on input", func(t *testing.T) {
		pr, pw := io.Pipe()
		t.Cleanup(func() { pr.Close(); pw.Close() })

		sleeper, _, _ := newTestSleeper(newFakeClock())
		sleeper.InputReader = pr // never written to, would block forever on its own

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() { done <- sleeper.WaitForUser(ctx) }()

		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("want context.Canceled, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("WaitForUser blocked instead of returning on cancellation")
		}
	})
}

func TestDefaultSleeperNotificationFailure(t *testing.T) {
	t.Run("a notification failure is reported on stderr, not mixed into progress, and does not fail the phase", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, progress, errs := newTestSleeper(clock)
		sleeper.Notify = func(title, message string) error { return errors.New("no notification daemon") }

		err := sleeper.Work(context.Background(), 1)

		if err != nil {
			t.Fatalf("a notification failure should not fail the phase, got: %v", err)
		}
		if !strings.Contains(errs.String(), "no notification daemon") {
			t.Errorf("expected the notification error on stderr, got: %q", errs.String())
		}
		if strings.Contains(progress.String(), "no notification daemon") {
			t.Errorf("notification error leaked into progress output: %q", progress.String())
		}
	})
}

func TestDefaultSleeperPhaseCancellation(t *testing.T) {
	t.Run("Work stops without sending any alert or waiting for input when cancelled", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, _, _ := newTestSleeper(clock)

		notified, played := false, false
		sleeper.Notify = func(title, message string) error { notified = true; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { played = true; return nil }
		sleeper.InputReader = &blockingReader{} // would hang forever if reached

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := sleeper.Work(ctx, 25)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
		if notified {
			t.Error("did not expect a notification to be sent for a cancelled phase")
		}
		if played {
			t.Error("did not expect a sound to be played for a cancelled phase")
		}
	})
}

func TestDefaultSleeperAlerts(t *testing.T) {
	t.Run("both enabled: notify and sound are each called exactly once", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		var notifyCalls, soundCalls int
		sleeper.Notify = func(title, message string) error { notifyCalls++; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { soundCalls++; return nil }
		sleeper.NotifyEnabled = true
		sleeper.SoundEnabled = true

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notifyCalls != 1 {
			t.Errorf("want 1 notify call, got %d", notifyCalls)
		}
		if soundCalls != 1 {
			t.Errorf("want 1 sound call, got %d", soundCalls)
		}
	})

	t.Run("only the notification is enabled", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		var notifyCalls, soundCalls int
		sleeper.Notify = func(title, message string) error { notifyCalls++; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { soundCalls++; return nil }
		sleeper.NotifyEnabled = true
		sleeper.SoundEnabled = false

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notifyCalls != 1 {
			t.Errorf("want the notification to be sent, got %d calls", notifyCalls)
		}
		if soundCalls != 0 {
			t.Errorf("want the sound to be skipped, got %d calls", soundCalls)
		}
	})

	t.Run("only the sound is enabled", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		var notifyCalls, soundCalls int
		sleeper.Notify = func(title, message string) error { notifyCalls++; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { soundCalls++; return nil }
		sleeper.NotifyEnabled = false
		sleeper.SoundEnabled = true

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notifyCalls != 0 {
			t.Errorf("want the notification to be skipped, got %d calls", notifyCalls)
		}
		if soundCalls != 1 {
			t.Errorf("want the sound to be played, got %d calls", soundCalls)
		}
	})

	t.Run("both disabled: neither alert is called", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		var notifyCalls, soundCalls int
		sleeper.Notify = func(title, message string) error { notifyCalls++; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { soundCalls++; return nil }
		sleeper.NotifyEnabled = false
		sleeper.SoundEnabled = false

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notifyCalls != 0 || soundCalls != 0 {
			t.Errorf("want no alerts, got notifyCalls=%d soundCalls=%d", notifyCalls, soundCalls)
		}
	})

	t.Run("a notification failure does not prevent the sound alert", func(t *testing.T) {
		sleeper, _, errs := newTestSleeper(newFakeClock())
		soundCalled := false
		sleeper.Notify = func(title, message string) error { return errors.New("no notification daemon") }
		sleeper.PlaySound = func(ctx context.Context) error { soundCalled = true; return nil }

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("a notification failure should not fail the phase, got: %v", err)
		}
		if !soundCalled {
			t.Error("expected the sound alert to still play after a notification failure")
		}
		if !strings.Contains(errs.String(), "no notification daemon") {
			t.Errorf("expected the notification error on stderr, got: %q", errs.String())
		}
	})

	t.Run("a sound failure does not invalidate the notification that already succeeded", func(t *testing.T) {
		sleeper, _, errs := newTestSleeper(newFakeClock())
		notifyCalled := false
		sleeper.Notify = func(title, message string) error { notifyCalled = true; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { return errors.New("no audio device") }

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("a sound failure should not fail the phase, got: %v", err)
		}
		if !notifyCalled {
			t.Error("expected the notification to have been sent")
		}
		if !strings.Contains(errs.String(), "no audio device") {
			t.Errorf("expected the sound error on stderr, got: %q", errs.String())
		}
	})

	t.Run("alert warnings never reach the progress writer", func(t *testing.T) {
		sleeper, progress, _ := newTestSleeper(newFakeClock())
		sleeper.Notify = func(title, message string) error { return errors.New("notify-boom") }
		sleeper.PlaySound = func(ctx context.Context) error { return errors.New("sound-boom") }

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(progress.String(), "boom") {
			t.Errorf("alert warnings leaked into progress output: %q", progress.String())
		}
	})

	t.Run("the sound receives the phase's context, so it can react to cancellation", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		var gotCtx context.Context
		sleeper.PlaySound = func(ctx context.Context) error { gotCtx = ctx; return nil }

		phaseCtx := context.Background()
		if err := sleeper.Work(phaseCtx, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotCtx != phaseCtx {
			t.Error("expected PlaySound to receive the same context the phase was run with")
		}
	})

	t.Run("a sound cancelled mid-playback is not reported as a warning", func(t *testing.T) {
		sleeper, _, errs := newTestSleeper(newFakeClock())
		sleeper.PlaySound = func(ctx context.Context) error { return context.Canceled }

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("a cancelled sound must not fail the phase on its own, got: %v", err)
		}
		if errs.Len() != 0 {
			t.Errorf("did not expect a warning for a cancelled sound, got: %q", errs.String())
		}
	})
}

// blockingReader blocks forever; used to prove a code path never reaches
// WaitForUser without needing a real terminal.
type blockingReader struct{}

func (*blockingReader) Read([]byte) (int, error) {
	select {}
}

// runCountdown runs runInteractiveCountdown in a goroutine and waits for
// its result, failing fast (instead of hanging the suite) if it never
// returns. The fake clock makes ticks resolve instantly, so a command sent
// on input genuinely races the ticker in Go's select -- that race is only
// over *which loop iteration* notices the command, never over the
// eventual outcome, so asserting on the final result (not on exactly how
// many ticks preceded it) is what these tests check.
func runCountdown(t *testing.T, sleeper *DefaultSleeper, ctx context.Context, minutes int, label string, input <-chan byte) (phaseResult, error) {
	t.Helper()
	type outcome struct {
		result phaseResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		r, err := sleeper.runInteractiveCountdown(ctx, minutes, label, input)
		done <- outcome{r, err}
	}()

	select {
	case o := <-done:
		return o.result, o.err
	case <-time.After(2 * time.Second):
		t.Fatal("runInteractiveCountdown did not return in time")
		return 0, nil
	}
}

// runWork calls sleeper.Work in a goroutine and waits for it, failing fast
// instead of hanging the suite if it never returns (e.g. a regression that
// makes a skipped phase fall through to WaitForUser against a reader that
// will never produce '\n').
func runWork(t *testing.T, sleeper *DefaultSleeper, ctx context.Context, minutes int) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- sleeper.Work(ctx, minutes) }()

	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Work did not return in time")
		return nil
	}
}

func TestDefaultSleeperInteractiveCountdown(t *testing.T) {
	t.Run("skip while running ends the phase as skipped, not completed", func(t *testing.T) {
		sleeper, progress, _ := newTestSleeper(newFakeClock())
		input := make(chan byte, 4)
		input <- 's'

		result, err := runCountdown(t, sleeper, context.Background(), 25, "Work", input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != phaseSkipped {
			t.Fatalf("result = %v, want phaseSkipped", result)
		}
		if !strings.Contains(progress.String(), "skipped") {
			t.Errorf("expected short skip feedback in output, got: %q", progress.String())
		}
	})

	t.Run("pause then resume then skip: the phase still ends as skipped, not completed or double-transitioned", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		input := make(chan byte, 4)
		input <- 'p' // pause
		input <- 'p' // resume
		input <- 's' // skip

		result, err := runCountdown(t, sleeper, context.Background(), 1, "Work", input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != phaseSkipped {
			t.Fatalf("result = %v, want phaseSkipped", result)
		}
	})

	t.Run("Ctrl+C (0x03) cancels the countdown", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		input := make(chan byte, 4)
		input <- 0x03

		result, err := runCountdown(t, sleeper, context.Background(), 25, "Work", input)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
		if result != phaseCancelled {
			t.Fatalf("result = %v, want phaseCancelled", result)
		}
	})

	t.Run("cancelling the parent context (not via keypress) also ends the countdown", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := runCountdown(t, sleeper, ctx, 25, "Work", make(chan byte))

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
		if result != phaseCancelled {
			t.Fatalf("result = %v, want phaseCancelled", result)
		}
	})

	t.Run("an unrecognized key is ignored, and a later skip still works", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		input := make(chan byte, 4)
		input <- 'x' // not a command
		input <- 's'

		result, err := runCountdown(t, sleeper, context.Background(), 25, "Work", input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != phaseSkipped {
			t.Fatalf("result = %v, want phaseSkipped", result)
		}
	})

	t.Run("a closed input channel (EOF) does not stop the countdown from completing normally", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		input := make(chan byte)
		close(input)

		result, err := runCountdown(t, sleeper, context.Background(), 1, "Work", input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != phaseCompleted {
			t.Fatalf("result = %v, want phaseCompleted", result)
		}
	})

	t.Run("no alerts are sent for a skipped phase, and WaitForUser is never reached", func(t *testing.T) {
		// The pipe writer sends exactly one 's' and then leaves the pipe
		// open without closing or writing again: if the skip result ever
		// stopped short-circuiting to WaitForUser, that call would block
		// forever reading from this same pipe, and runWork below would
		// time out instead of returning -- so this test fails loudly
		// rather than hanging if that regresses.
		sleeper, _, _ := newTestSleeper(newFakeClock())
		sleeper.Interactive = true
		sleeper.EnterRawMode = func() (func(), error) { return func() {}, nil }

		pr, pw := io.Pipe()
		t.Cleanup(func() { pr.Close(); pw.Close() })
		sleeper.InputReader = pr
		go func() { pw.Write([]byte("s")) }()

		notified, played := false, false
		sleeper.Notify = func(title, message string) error { notified = true; return nil }
		sleeper.PlaySound = func(ctx context.Context) error { played = true; return nil }

		if err := runWork(t, sleeper, context.Background(), 25); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notified || played {
			t.Errorf("did not expect alerts for a skipped phase, notified=%v played=%v", notified, played)
		}
	})

	t.Run("when raw mode is unavailable, the phase falls back to the plain countdown and still completes", func(t *testing.T) {
		sleeper, _, _ := newTestSleeper(newFakeClock())
		sleeper.Interactive = true
		sleeper.EnterRawMode = func() (func(), error) { return nil, errors.New("not a terminal") }

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("when not interactive, the countdown behaves exactly as the plain (non-interactive) one", func(t *testing.T) {
		sleeper, progress, _ := newTestSleeper(newFakeClock())
		sleeper.Interactive = false

		if err := sleeper.Work(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(progress.String(), "[p] Pause") {
			t.Errorf("did not expect interactive controls hint when not interactive, got: %q", progress.String())
		}
	})
}
