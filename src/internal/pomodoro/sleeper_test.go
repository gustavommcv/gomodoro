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
	t.Run("Work stops without sending a notification or waiting for input when cancelled", func(t *testing.T) {
		clock := newFakeClock()
		sleeper, _, _ := newTestSleeper(clock)

		notified := false
		sleeper.Notify = func(title, message string) error { notified = true; return nil }
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
	})
}

// blockingReader blocks forever; used to prove a code path never reaches
// WaitForUser without needing a real terminal.
type blockingReader struct{}

func (*blockingReader) Read([]byte) (int, error) {
	select {}
}
