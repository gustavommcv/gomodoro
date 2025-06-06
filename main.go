package main

import (
	"fmt"
	"time"

	"example.com/pomodoro"
)

type DefaultSleeper struct{}

func (d *DefaultSleeper) Work(minutes int) {
	fmt.Println("Working")
	time.Sleep(2 * time.Minute)
}

func (d *DefaultSleeper) ShortBreak(minutes int) {
	fmt.Println("Short Break")
	time.Sleep(pomodoro.DEFAULT_SHORT_BREAK)
	time.Sleep(pomodoro.DEFAULT_SHORT_BREAK * time.Minute)
}

func (d *DefaultSleeper) LongBreak(minutes int) {
	fmt.Println("Long Break")
	time.Sleep(pomodoro.DEFAULT_LONG_BREAK)
	time.Sleep(pomodoro.DEFAULT_LONG_BREAK * time.Minute)
}

func main() {
	sleeper := DefaultSleeper{}

	pomodoro.Pomodoro(&sleeper)
}
