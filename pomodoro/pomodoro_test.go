package pomodoro

import (
	"reflect"
	"testing"
)

type SpySleeper struct {
	CallCount CallCount
}

type CallCount struct {
	Work       int
	ShortBreak int
	LongBreak  int
}

func (s *SpySleeper) Work() {
	s.CallCount.Work++
}

func (s *SpySleeper) ShortBreak() {
	s.CallCount.ShortBreak++
}

func (s *SpySleeper) LongBreak() {
	s.CallCount.LongBreak++
}

func TestPomodoro(t *testing.T) {
	t.Run("it should use default params when no args are supplied", func(t *testing.T) {})

	t.Run("it should throw a error when invalid args are supplid", func(t *testing.T) {})

	t.Run("it should run 4x work/short break before taking 1 long break", func(t *testing.T) {
		sleeper := SpySleeper{}
		Pomodoro(&sleeper)

		want := CallCount{Work: 4, ShortBreak: 4, LongBreak: 1}
		got := sleeper.CallCount

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expect %v, got %v", want, got)
		}
	})
}
