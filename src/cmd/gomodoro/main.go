package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

func main() {
	work := flag.Int("work", pomodoro.DEFAULT_WORK_TIMER, "Work duration in minutes")
	sbreak := flag.Int("sbreak", pomodoro.DEFAULT_SHORT_BREAK, "Short break duration in minutes")
	lbreak := flag.Int("lbreak", pomodoro.DEFAULT_LONG_BREAK, "Long break duration in minutes")

	// Custom usage message
	flag.Usage = func() {
		printCustomUsage()
	}

	flag.Parse()

	runPomodoro(*work, *sbreak, *lbreak)
}

func runPomodoro(work, sbreak, lbreak int) {
	sleeper := pomodoro.DefaultSleeper{ProgressWriter: os.Stdout}
	pomodoro.Pomodoro(&sleeper, work, sbreak, lbreak)
}

func printCustomUsage() {
	fmt.Printf(`
Gomodoro - Pomodoro Timer CLI

Usage:
  %s [flags]

Flags:
  -work int      Work duration in minutes (default %d)
  -sbreak int    Short break duration in minutes (default %d) 
  -lbreak int    Long break duration in minutes (default %d)

Examples:
  # Start with default values (%d/%d/%d)
  %s
  
  # Custom durations
  %s -work=50 -sbreak=10
  %s -work=25 -sbreak=5 -lbreak=20
`,
		os.Args[0],
		pomodoro.DEFAULT_WORK_TIMER, pomodoro.DEFAULT_SHORT_BREAK, pomodoro.DEFAULT_LONG_BREAK,
		pomodoro.DEFAULT_WORK_TIMER, pomodoro.DEFAULT_SHORT_BREAK, pomodoro.DEFAULT_LONG_BREAK,
		os.Args[0],
		os.Args[0],
		os.Args[0])
}
