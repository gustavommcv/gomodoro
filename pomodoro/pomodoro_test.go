package pomodoro

import (
	"reflect"
	"testing"
)

type SpySleeper struct {
	CallCount      CallCount
	WorkTime       int
	ShortBreakTime int
	LongBreakTime  int
	Calls          []string
}

type CallCount struct {
	Work       int
	ShortBreak int
	LongBreak  int
}

func (s *SpySleeper) Work(minutes int) {
	if s.WorkTime == 0 {
		s.WorkTime = minutes
	}

	s.CallCount.Work++

	s.Calls = append(s.Calls, work)
}

func (s *SpySleeper) ShortBreak(minutes int) {
	if s.ShortBreakTime == 0 {
		s.ShortBreakTime = minutes
	}

	s.CallCount.ShortBreak++

	s.Calls = append(s.Calls, shortBreak)
}

func (s *SpySleeper) LongBreak(minutes int) {
	if s.LongBreakTime == 0 {
		s.LongBreakTime = minutes
	}

	s.CallCount.LongBreak++

	s.Calls = append(s.Calls, longBreak)
}

func TestPomodoro(t *testing.T) {
	t.Run("it should use default params when no args are supplied", func(t *testing.T) {
		sleeper := SpySleeper{}

		Pomodoro(&sleeper, 0, 0, 0)

		want := struct {
			WorkTime       int
			ShortBreakTime int
			LongBreakTime  int
		}{
			WorkTime:       DEFAULT_WORK_TIMER,
			ShortBreakTime: DEFAULT_SHORT_BREAK,
			LongBreakTime:  DEFAULT_LONG_BREAK,
		}
		got := struct {
			WorkTime       int
			ShortBreakTime int
			LongBreakTime  int
		}{
			WorkTime:       sleeper.WorkTime,
			ShortBreakTime: sleeper.ShortBreakTime,
			LongBreakTime:  sleeper.LongBreakTime,
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})

	t.Run("it should use custom params when args are supplied", func(t *testing.T) {
		sleeper := SpySleeper{}
		workTimer := 10
		shortBreakTimer := 15
		longBreakTimer := 5

		Pomodoro(&sleeper, workTimer, shortBreakTimer, longBreakTimer)

			want := struct {
			WorkTime       int
			ShortBreakTime int
			LongBreakTime  int
		}{
			WorkTime:       workTimer,
			ShortBreakTime: shortBreakTimer,
			LongBreakTime: longBreakTimer,
		}
		got := struct {
			WorkTime       int
			ShortBreakTime int
			LongBreakTime  int
		}{
			WorkTime:       sleeper.WorkTime,
			ShortBreakTime: sleeper.ShortBreakTime,
			LongBreakTime:  sleeper.LongBreakTime,
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})

	// Already done by flag module
	// t.Run("it should throw a error when invalid args are supplid", func(t *testing.T) {})

	t.Run("it should run 4x work and 3x short break before taking 1 long break", func(t *testing.T) {
		sleeper := SpySleeper{}

		Pomodoro(&sleeper, 0, 0, 0)

		want := CallCount{Work: 4, ShortBreak: 3, LongBreak: 1}
		got := sleeper.CallCount

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})

	t.Run("Testing the order of the pomodoro call", func(t *testing.T) {
		sleeper := SpySleeper{Calls: make([]string, 0)}

		Pomodoro(&sleeper, 4, 3, 1)

		want := []string{
			work,
			shortBreak,
			work,
			shortBreak,
			work,
			shortBreak,
			work,
			longBreak,
		}

		if !reflect.DeepEqual(want, sleeper.Calls) {
			t.Errorf("wanted calls %v got %v", want, sleeper.Calls)
		}

	})
}
