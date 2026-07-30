package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

// options holds the resolved CLI configuration for one Pomodoro lap.
type options struct {
	work   int
	sbreak int
	lbreak int
}

// parseArgs accepts both the direct flag syntax (`gomodoro -work=25`) and
// the `start` subcommand (`gomodoro start -work=25`); the two are
// equivalent. It never writes to a global flag set or os.Args, so it can be
// tested with arbitrary argument slices. Usage text is not printed here:
// the caller decides where help (stdout) and errors (stderr) go.
func parseArgs(args []string) (options, error) {
	rest := args
	if len(rest) > 0 && rest[0] == "start" {
		rest = rest[1:]
	}

	fs := flag.NewFlagSet("gomodoro", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {} // run() decides where/whether to print usage

	work := fs.Int("work", pomodoro.DefaultWorkMinutes, "Work duration in minutes")
	sbreak := fs.Int("sbreak", pomodoro.DefaultShortBreakMinutes, "Short break duration in minutes")
	lbreak := fs.Int("lbreak", pomodoro.DefaultLongBreakMinutes, "Long break duration in minutes")

	if err := fs.Parse(rest); err != nil {
		return options{}, err
	}

	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected argument(s): %s", strings.Join(fs.Args(), " "))
	}

	opts := options{work: *work, sbreak: *sbreak, lbreak: *lbreak}
	if err := opts.validate(); err != nil {
		return options{}, err
	}

	return opts, nil
}

func (o options) validate() error {
	if o.work <= 0 {
		return fmt.Errorf("-work must be greater than zero, got %d", o.work)
	}
	if o.sbreak <= 0 {
		return fmt.Errorf("-sbreak must be greater than zero, got %d", o.sbreak)
	}
	if o.lbreak <= 0 {
		return fmt.Errorf("-lbreak must be greater than zero, got %d", o.lbreak)
	}
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `Gomodoro - Pomodoro Timer CLI

Runs one lap: four work sessions, a short break after each of the first
three, and a long break after the fourth. Then it exits.

Usage:
  gomodoro [flags]
  gomodoro start [flags]

Flags:
  -work int      Work duration in minutes (default %d)
  -sbreak int    Short break duration in minutes (default %d)
  -lbreak int    Long break duration in minutes (default %d)

Examples:
  gomodoro
  gomodoro -work=50 -sbreak=10
  gomodoro start
  gomodoro start -work=25 -sbreak=5 -lbreak=20

Press Ctrl+C at any time to stop.
`,
		pomodoro.DefaultWorkMinutes, pomodoro.DefaultShortBreakMinutes, pomodoro.DefaultLongBreakMinutes)
}
