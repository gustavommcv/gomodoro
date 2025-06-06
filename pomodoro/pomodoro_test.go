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
}

func (s *SpySleeper) ShortBreak(minutes int) {
	if s.ShortBreakTime == 0 {
		s.ShortBreakTime = minutes
	}

	s.CallCount.ShortBreak++
}

func (s *SpySleeper) LongBreak(minutes int) {
	if s.LongBreakTime == 0 {
		s.LongBreakTime = minutes
	}

	s.CallCount.LongBreak++
}

func TestPomodoro(t *testing.T) {
	t.Run("it should use default params when no args are supplied", func(t *testing.T) {
		sleeper := SpySleeper{}

		Pomodoro(&sleeper, 0, 0, 0)

		want := SpySleeper{CallCount: sleeper.CallCount, WorkTime: DEFAULT_WORK_TIMER, ShortBreakTime: DEFAULT_SHORT_BREAK, LongBreakTime: DEFAULT_LONG_BREAK}
		got := sleeper

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

		want := SpySleeper{CallCount: sleeper.CallCount, WorkTime: workTimer, ShortBreakTime: shortBreakTimer, LongBreakTime: longBreakTimer}
		got := sleeper

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})

	// Already done by flag module
	// t.Run("it should throw a error when invalid args are supplid", func(t *testing.T) {})

	t.Run("it should run 4x work/short break before taking 1 long break", func(t *testing.T) {
		sleeper := SpySleeper{}
		Pomodoro(&sleeper, 0, 0, 0)

		want := CallCount{Work: 4, ShortBreak: 4, LongBreak: 1}
		got := sleeper.CallCount

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})
}
