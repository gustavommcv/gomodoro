package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"example.com/pomodoro"
	"example.com/progressbar"
)

type DefaultSleeper struct {
	progressWriter io.Writer
}

func (d *DefaultSleeper) Work(minutes int) {
	fmt.Printf("\nWorking for %d minutes\n", minutes)
	d.showProgress(minutes, "Work")
}

func (d *DefaultSleeper) ShortBreak(minutes int) {
	fmt.Printf("\nShort Break for %d minutes\n", minutes)
	d.showProgress(minutes, "Short Break")
}

func (d *DefaultSleeper) LongBreak(minutes int) {
	fmt.Printf("\nLong Break for %d minutes\n", minutes)
	d.showProgress(minutes, "Long Break")
}

func (d *DefaultSleeper) showProgress(totalMinutes int, label string) {
	const resolution = 60
	totalSeconds := totalMinutes * 60

	for i := 1; i <= totalSeconds; i++ {
		fmt.Fprint(d.progressWriter, "\r")

		progressbar.ProgressBar(d.progressWriter, float64(i), float64(totalSeconds))

		remaining := totalSeconds - i
		fmt.Fprintf(d.progressWriter, " - %s: %02d:%02d remaining",
			label,
			remaining/60,
			remaining%60)

		time.Sleep(1 * time.Second)
	}
	fmt.Fprintln(d.progressWriter)
}

func main() {

	pomodoroPrinter := pomodoro.PomodoroOperations{Calls: make([]string, 0)}
	workDuration := flag.Int("work", 0, "Work duration in minutes")
	shortBreakDuration := flag.Int("sbreak", 0, "Short break duration in minutes")
	longBreakDuration := flag.Int("lbreak", 0, "Long break duration in minutes")

	flag.Parse()

	sleeper := DefaultSleeper{progressWriter: os.Stdout}

	pomodoro.Pomodoro(&pomodoroPrinter, &sleeper, *workDuration, *shortBreakDuration, *longBreakDuration)
}
