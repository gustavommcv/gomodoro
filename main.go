package main

import (
	"flag"
	"fmt"
	"time"

	"example.com/pomodoro"
)

type DefaultSleeper struct{}

func (d *DefaultSleeper) Work(minutes int) {
	fmt.Println("Working")
	fmt.Println(minutes)
	time.Sleep(time.Duration(minutes) * time.Minute)
}

func (d *DefaultSleeper) ShortBreak(minutes int) {
	fmt.Println("Short Break")
	time.Sleep(time.Duration(minutes) * time.Minute)
}

func (d *DefaultSleeper) LongBreak(minutes int) {
	fmt.Println("Long Break")
	time.Sleep(time.Duration(minutes) * time.Minute)
}

func main() {
	workDuration := flag.Int("work", 0, "Work duration in minutes")
	shortBreakDuration := flag.Int("sbreak", 0, "Short break duration in minutes")
	longBreakDuration := flag.Int("lbreak", 0, "Long break duration in minutes")

	flag.Parse()

	sleeper := DefaultSleeper{}

	pomodoro.Pomodoro(&sleeper, *workDuration, *shortBreakDuration, *longBreakDuration)
}
