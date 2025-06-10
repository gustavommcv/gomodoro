package pomodoro

import (
	"fmt"
	"io"
	"time"

	"github.com/gustavommcv/gomodoro/src/pkg/notify"
	"github.com/gustavommcv/gomodoro/src/pkg/progressbar"
)

const DEFAULT_WORK_TIMER int = 25
const DEFAULT_SHORT_BREAK int = 5
const DEFAULT_LONG_BREAK int = 15
const CICLES_PER_LAP int = 4

type Sleeper interface {
	Work(minutes int)
	ShortBreak(minutes int)
	LongBreak(minutes int)
	WaitForUser()
}

type DefaultSleeper struct {
	ProgressWriter io.Writer
	InputReader    io.Reader
}

// TODO
// Pause execution when a notification is sent
// Need to wait for user interaction
func (d *DefaultSleeper) Work(minutes int) {
	fmt.Printf("\nWorking for %d minutes\n", minutes)
	d.showProgress(minutes, "Work")
	notify.Notify("Work Finished, press enter in the terminal to continue")
	d.WaitForUser()
}

func (d *DefaultSleeper) ShortBreak(minutes int) {
	fmt.Printf("\nShort Break for %d minutes\n", minutes)
	d.showProgress(minutes, "Short Break")
	notify.Notify("Short break Finished, press enter in the terminal to continue")
	d.WaitForUser()
}

func (d *DefaultSleeper) LongBreak(minutes int) {
	fmt.Printf("\nLong Break for %d minutes\n", minutes)
	d.showProgress(minutes, "Long Break")
	notify.Notify("Long break Finished, press enter in the terminal to continue")
	d.WaitForUser()
}

func (d *DefaultSleeper) WaitForUser() {
	fmt.Fprint(d.ProgressWriter, "\nPress Enter to continue...")

	buf := make([]byte, 1)
	for {
		_, err := d.InputReader.Read(buf)
		if err != nil || buf[0] == '\n' {
			break
		}
	}
}

func (d *DefaultSleeper) showProgress(totalMinutes int, label string) {
	const resolution = 60
	totalSeconds := totalMinutes * 60

	for i := 1; i <= totalSeconds; i++ {
		fmt.Fprint(d.ProgressWriter, "\r")

		progressbar.ProgressBar(d.ProgressWriter, float64(i), float64(totalSeconds))

		remaining := totalSeconds - i
		fmt.Fprintf(d.ProgressWriter, " - %s: %02d:%02d remaining",
			label,
			remaining/60,
			remaining%60)

		time.Sleep(1 * time.Second)
	}
	fmt.Fprintln(d.ProgressWriter)
}

func Pomodoro(sleeper Sleeper, work, shortBreak, longBreak int) {
	if work == 0 {
		work = DEFAULT_WORK_TIMER
	}

	if shortBreak == 0 {
		shortBreak = DEFAULT_SHORT_BREAK
	}

	if longBreak == 0 {
		longBreak = DEFAULT_LONG_BREAK
	}

	for actualCicle := 1; actualCicle <= CICLES_PER_LAP; actualCicle++ {
		sleeper.Work(work)
		if actualCicle == CICLES_PER_LAP {
			sleeper.LongBreak(longBreak)
		} else {
			sleeper.ShortBreak(shortBreak)
		}
	}
}
