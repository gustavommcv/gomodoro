package pomodoro

import (
	"reflect"
	"testing"
)

const WORK string = "work"
const SHORT_BREAK string = "shortBreak"
const LONG_BREAK string = "longBreak"

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

func (s *SpySleeper) WaitForUser() {}

func (s *SpySleeper) Work(minutes int) {
	if s.WorkTime == 0 {
		s.WorkTime = minutes
	}

	s.CallCount.Work++
	s.Calls = append(s.Calls, WORK)
}

func (s *SpySleeper) ShortBreak(minutes int) {
	if s.ShortBreakTime == 0 {
		s.ShortBreakTime = minutes
	}

	s.CallCount.ShortBreak++
	s.Calls = append(s.Calls, SHORT_BREAK)
}

func (s *SpySleeper) LongBreak(minutes int) {
	if s.LongBreakTime == 0 {
		s.LongBreakTime = minutes
	}

	s.CallCount.LongBreak++
	s.Calls = append(s.Calls, LONG_BREAK)
}

func TestPomodoro(t *testing.T) {
	t.Run("it should use default params when no flags are supplied", func(t *testing.T) {
		sleeper := SpySleeper{}

		Pomodoro(&sleeper, 0, 0, 0)

		want := [...]int{DEFAULT_WORK_TIMER, DEFAULT_SHORT_BREAK, DEFAULT_LONG_BREAK}
		got := [...]int{sleeper.WorkTime, sleeper.ShortBreakTime, sleeper.LongBreakTime}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("it should use custom params when flags are supplied", func(t *testing.T) {
		sleeper := SpySleeper{}
		workTimer := 10
		shortBreakTimer := 15
		longBreakTimer := 5

		Pomodoro(&sleeper, workTimer, shortBreakTimer, longBreakTimer)

		want := [...]int{workTimer, shortBreakTimer, longBreakTimer}
		got := [...]int{sleeper.WorkTime, sleeper.ShortBreakTime, sleeper.LongBreakTime}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	// Already done by flag module
	// t.Run("it should throw a error when invalid flags are supplid", func(t *testing.T) {})

	t.Run("it should run 4x work and 3x short break before taking 1 long break", func(t *testing.T) {
		sleeper := SpySleeper{}

		Pomodoro(&sleeper, 0, 0, 0)

		want := CallCount{Work: 4, ShortBreak: 3, LongBreak: 1}
		got := sleeper.CallCount

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("should execute pomodoro cycles in correct order: work -> short breaks -> long break after last cycle", func(t *testing.T) {
		sleeper := SpySleeper{Calls: make([]string, 0)}

		Pomodoro(&sleeper, 4, 3, 1)

		want := []string{
			WORK,
			SHORT_BREAK,
			WORK,
			SHORT_BREAK,
			WORK,
			SHORT_BREAK,
			WORK,
			LONG_BREAK,
		}

		got := sleeper.Calls

		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
}
