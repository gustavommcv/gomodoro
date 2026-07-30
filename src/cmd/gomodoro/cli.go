package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

// defaultNotifyEnabled and defaultSoundEnabled are the application defaults
// used while no config file exists. They are the bottom of the resolution
// order in resolveOptions.
const (
	defaultNotifyEnabled = true
	defaultSoundEnabled  = true
)

// optionalBool is a tri-state boolean for a CLI flag: unset, explicitly
// true, or explicitly false. Plain bool cannot represent "not provided",
// which a future config file needs: an explicit flag must override the
// config value, but an absent flag must not override it with a false
// zero-value. It implements flag.Value (and the flag package's unexported
// boolFlag interface via IsBoolFlag) so it plugs into flag.FlagSet like any
// other flag type.
type optionalBool struct {
	set   bool
	value bool
}

func (b *optionalBool) String() string {
	if b == nil || !b.set {
		return ""
	}
	return strconv.FormatBool(b.value)
}

func (b *optionalBool) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("invalid boolean value %q", s)
	}
	b.set = true
	b.value = v
	return nil
}

// IsBoolFlag lets the flag package treat -notify (with no =value) as
// -notify=true, matching the ergonomics of flag.Bool.
func (b *optionalBool) IsBoolFlag() bool { return true }

// resolve returns the flag's explicit value if it was set, otherwise def.
func (b optionalBool) resolve(def bool) bool {
	if !b.set {
		return def
	}
	return b.value
}

// options holds the resolved CLI configuration for one Pomodoro lap.
type options struct {
	work   int
	sbreak int
	lbreak int
	notify bool
	sound  bool
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

	var notify, sound optionalBool
	fs.Var(&notify, "notify", "Show a desktop notification at the end of each phase")
	fs.Var(&sound, "sound", "Play a short alert sound at the end of each phase")

	if err := fs.Parse(rest); err != nil {
		return options{}, err
	}

	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected argument(s): %s", strings.Join(fs.Args(), " "))
	}

	opts := resolveOptions(*work, *sbreak, *lbreak, notify, sound)
	if err := opts.validate(); err != nil {
		return options{}, err
	}

	return opts, nil
}

// resolveOptions is the single place that decides effective values from
// what the CLI received. Today that's just "explicit flag, else default";
// this is the seam a future config file slots into (explicit flag > config
// value > default) without parseArgs itself having to change.
func resolveOptions(work, sbreak, lbreak int, notify, sound optionalBool) options {
	return options{
		work:   work,
		sbreak: sbreak,
		lbreak: lbreak,
		notify: notify.resolve(defaultNotifyEnabled),
		sound:  sound.resolve(defaultSoundEnabled),
	}
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
  -notify bool   Show a desktop notification at the end of each phase (default true)
  -sound bool    Play a short alert sound at the end of each phase (default true)

Examples:
  gomodoro
  gomodoro -work=50 -sbreak=10
  gomodoro start
  gomodoro start -work=25 -sbreak=5 -lbreak=20
  gomodoro start -notify=false
  gomodoro start -sound=false
  gomodoro start -notify=false -sound=false

Notification and sound are independent: disabling one does not disable the
other.

While a phase is running (in a real terminal only):
  p    Pause / resume
  s    Skip the current phase
  Ctrl+C   Quit

Press Ctrl+C at any time to stop.
`,
		pomodoro.DefaultWorkMinutes, pomodoro.DefaultShortBreakMinutes, pomodoro.DefaultLongBreakMinutes)
}
