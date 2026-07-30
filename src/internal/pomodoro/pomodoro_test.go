package pomodoro

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

const Work string = "work"
const ShortBreak string = "shortBreak"
const LongBreak string = "longBreak"

type SpySleeper struct {
	CallCount      CallCount
	WorkTime       int
	ShortBreakTime int
	LongBreakTime  int
	Calls          []string

	// FailAfter makes the phase call at this 1-based index (across all calls)
	// return the given error instead of succeeding. Zero disables the failure.
	FailAfter int
	FailErr   error
}

type CallCount struct {
	Work       int
	ShortBreak int
	LongBreak  int
}

func (s *SpySleeper) WaitForUser(ctx context.Context) error { return nil }

func (s *SpySleeper) totalCalls() int {
	return s.CallCount.Work + s.CallCount.ShortBreak + s.CallCount.LongBreak
}

func (s *SpySleeper) shouldFail() bool {
	return s.FailAfter > 0 && s.totalCalls()+1 == s.FailAfter
}

func (s *SpySleeper) Work(ctx context.Context, minutes int) error {
	if s.WorkTime == 0 {
		s.WorkTime = minutes
	}

	fail := s.shouldFail()
	s.CallCount.Work++
	s.Calls = append(s.Calls, Work)

	if fail {
		return s.FailErr
	}
	return nil
}

func (s *SpySleeper) ShortBreak(ctx context.Context, minutes int) error {
	if s.ShortBreakTime == 0 {
		s.ShortBreakTime = minutes
	}

	fail := s.shouldFail()
	s.CallCount.ShortBreak++
	s.Calls = append(s.Calls, ShortBreak)

	if fail {
		return s.FailErr
	}
	return nil
}

func (s *SpySleeper) LongBreak(ctx context.Context, minutes int) error {
	if s.LongBreakTime == 0 {
		s.LongBreakTime = minutes
	}

	fail := s.shouldFail()
	s.CallCount.LongBreak++
	s.Calls = append(s.Calls, LongBreak)

	if fail {
		return s.FailErr
	}
	return nil
}

func TestPomodoro(t *testing.T) {
	ctx := context.Background()

	t.Run("it should use the durations it is given", func(t *testing.T) {
		sleeper := SpySleeper{}
		workTimer := 10
		shortBreakTimer := 15
		longBreakTimer := 5

		if err := Pomodoro(ctx, &sleeper, workTimer, shortBreakTimer, longBreakTimer); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := [...]int{workTimer, shortBreakTimer, longBreakTimer}
		got := [...]int{sleeper.WorkTime, sleeper.ShortBreakTime, sleeper.LongBreakTime}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("it should run 4x work and 3x short break before taking 1 long break", func(t *testing.T) {
		sleeper := SpySleeper{}

		if err := Pomodoro(ctx, &sleeper, DefaultWorkMinutes, DefaultShortBreakMinutes, DefaultLongBreakMinutes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := CallCount{Work: 4, ShortBreak: 3, LongBreak: 1}
		got := sleeper.CallCount

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("should execute pomodoro cycles in correct order: work -> short breaks -> long break after last cycle", func(t *testing.T) {
		sleeper := SpySleeper{Calls: make([]string, 0)}

		if err := Pomodoro(ctx, &sleeper, 4, 3, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []string{
			Work,
			ShortBreak,
			Work,
			ShortBreak,
			Work,
			ShortBreak,
			Work,
			LongBreak,
		}

		got := sleeper.Calls

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("it should reject a work duration of zero", func(t *testing.T) {
		sleeper := SpySleeper{}

		err := Pomodoro(ctx, &sleeper, 0, DefaultShortBreakMinutes, DefaultLongBreakMinutes)

		if err == nil {
			t.Fatal("expected an error for a zero work duration, got none")
		}
		if sleeper.totalCalls() != 0 {
			t.Errorf("expected no phase to run, got %d calls", sleeper.totalCalls())
		}
	})

	t.Run("it should reject a negative short break duration", func(t *testing.T) {
		sleeper := SpySleeper{}

		err := Pomodoro(ctx, &sleeper, DefaultWorkMinutes, -5, DefaultLongBreakMinutes)

		if err == nil {
			t.Fatal("expected an error for a negative short break duration, got none")
		}
		if sleeper.totalCalls() != 0 {
			t.Errorf("expected no phase to run, got %d calls", sleeper.totalCalls())
		}
	})

	t.Run("it should reject a negative long break duration", func(t *testing.T) {
		sleeper := SpySleeper{}

		err := Pomodoro(ctx, &sleeper, DefaultWorkMinutes, DefaultShortBreakMinutes, -1)

		if err == nil {
			t.Fatal("expected an error for a negative long break duration, got none")
		}
		if sleeper.totalCalls() != 0 {
			t.Errorf("expected no phase to run, got %d calls", sleeper.totalCalls())
		}
	})

	t.Run("it should stop immediately when a phase is cancelled", func(t *testing.T) {
		wantErr := errors.New("boom")
		sleeper := SpySleeper{FailAfter: 2, FailErr: wantErr} // fails on the 2nd phase call (first short break)

		err := Pomodoro(ctx, &sleeper, DefaultWorkMinutes, DefaultShortBreakMinutes, DefaultLongBreakMinutes)

		if !errors.Is(err, wantErr) {
			t.Fatalf("want error %v, got %v", wantErr, err)
		}
		if sleeper.totalCalls() != 2 {
			t.Errorf("expected the lap to stop after the failing phase, got %d calls", sleeper.totalCalls())
		}
	})
}
