package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gustavommcv/gomodoro/src/internal/config"
	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
)

// cliVersion is a plain, hand-maintained version string. There is no
// release/build pipeline yet (out of scope for this stage) -- --version
// exists so scripts and bug reports have something stable to reference.
const cliVersion = "0.1.0-dev"

// optionalBool is a tri-state boolean for a CLI flag: unset, explicitly
// true, or explicitly false. Plain bool cannot represent "not provided",
// which the config file layer needs: an explicit flag must override the
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

func (b optionalBool) resolve(def bool) bool {
	if !b.set {
		return def
	}
	return b.value
}

// optionalInt mirrors optionalBool for integer flags (-work/-sbreak/-lbreak),
// for exactly the same reason: distinguishing "not provided" from "provided,
// and it happens to match the default" is required for flag > config file >
// compiled default precedence to work.
type optionalInt struct {
	set   bool
	value int
}

func (o *optionalInt) String() string {
	if o == nil || !o.set {
		return ""
	}
	return strconv.Itoa(o.value)
}

func (o *optionalInt) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid integer value %q", s)
	}
	o.set = true
	o.value = v
	return nil
}

func (o optionalInt) resolve(def int) int {
	if !o.set {
		return def
	}
	return o.value
}

// options holds the resolved CLI configuration for one Pomodoro lap.
type options struct {
	work   int
	sbreak int
	lbreak int
	notify bool
	sound  bool
}

// extractConfigFlag pulls -config/--config out of args wherever it
// appears, since --config is documented to work both before and after the
// start subcommand (unlike -work/-sbreak/-lbreak/-notify/-sound, which are
// parsed together with/after start as before). It returns the flag's
// value (empty if absent) and the remaining arguments with the flag and
// its value removed, in their original relative order.
func extractConfigFlag(args []string) (value string, rest []string, err error) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-config" || a == "--config":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag needs an argument: %s", a)
			}
			value = args[i+1]
			i++
		case strings.HasPrefix(a, "-config="):
			value = strings.TrimPrefix(a, "-config=")
		case strings.HasPrefix(a, "--config="):
			value = strings.TrimPrefix(a, "--config=")
		default:
			rest = append(rest, a)
		}
	}
	return value, rest, nil
}

// hasHelpFlag reports whether args requests help, checked before any
// config file is loaded -- help must work even with a broken config.
func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			return true
		}
	}
	return false
}

// parseArgs accepts both the direct flag syntax (`gomodoro -work=25`) and
// the `start` subcommand (`gomodoro start -work=25`); the two are
// equivalent. base is the already-resolved config (Defaults() overlaid by
// a config file, if any) that unset flags fall back to -- this is the
// seam described in resolveOptions. It never writes to a global flag set
// or os.Args, so it can be tested with arbitrary argument slices. Usage
// text is not printed here: the caller decides where help (stdout) and
// errors (stderr) go.
func parseArgs(args []string, base config.Config) (options, error) {
	rest := args
	if len(rest) > 0 && rest[0] == "start" {
		rest = rest[1:]
	}

	fs := flag.NewFlagSet("gomodoro", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {} // run() decides where/whether to print usage

	var work, sbreak, lbreak optionalInt
	fs.Var(&work, "work", "Work duration in minutes")
	fs.Var(&sbreak, "sbreak", "Short break duration in minutes")
	fs.Var(&lbreak, "lbreak", "Long break duration in minutes")

	var notify, sound optionalBool
	fs.Var(&notify, "notify", "Show a desktop notification at the end of each phase")
	fs.Var(&sound, "sound", "Play a short alert sound at the end of each phase")

	if err := fs.Parse(rest); err != nil {
		return options{}, err
	}

	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected argument(s): %s", strings.Join(fs.Args(), " "))
	}

	opts := resolveOptions(work, sbreak, lbreak, notify, sound, base)
	if err := opts.validate(); err != nil {
		return options{}, err
	}

	return opts, nil
}

// resolveOptions is the single place that decides effective values:
// explicit flag > base (already config-file-overlaid) > nothing else.
// Because durations and alerts are both tri-state here, this one function
// implements the full "flag > config file > compiled default" precedence
// without parseArgs needing to know anything about that chain itself.
func resolveOptions(work, sbreak, lbreak optionalInt, notify, sound optionalBool, base config.Config) options {
	return options{
		work:   work.resolve(int(base.Timer.Work.Minutes())),
		sbreak: sbreak.resolve(int(base.Timer.ShortBreak.Minutes())),
		lbreak: lbreak.resolve(int(base.Timer.LongBreak.Minutes())),
		notify: notify.resolve(base.Alerts.Notify),
		sound:  sound.resolve(base.Alerts.Sound),
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
  gomodoro config path|init|check [--config path]
  gomodoro help | --help | -h
  gomodoro --version

Flags:
  -work int      Work duration in minutes (default %d)
  -sbreak int    Short break duration in minutes (default %d)
  -lbreak int    Long break duration in minutes (default %d)
  -notify bool   Show a desktop notification at the end of each phase (default true)
  -sound bool    Play a short alert sound at the end of each phase (default true)
  --config path  Use this config file instead of the default location
                 (works before or after "start"). GOMODORO_CONFIG does
                 the same thing via an environment variable, with lower
                 precedence than --config.

Examples:
  gomodoro
  gomodoro -work=50 -sbreak=10
  gomodoro start
  gomodoro start -work=25 -sbreak=5 -lbreak=20
  gomodoro start -notify=false
  gomodoro start -sound=false
  gomodoro --config ./my-config.lua start
  gomodoro start --config ./my-config.lua
  gomodoro config path
  gomodoro config init
  gomodoro config check

Precedence for -work/-sbreak/-lbreak/-notify/-sound: an explicit flag
always wins, then the config file (if any), then the compiled-in default.
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

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "gomodoro %s\n", cliVersion)
}

// runConfigCommand dispatches "gomodoro config <subcommand>". None of
// these subcommands start the timer, alerts, or raw mode.
func runConfigCommand(args []string, stdout, stderr io.Writer, getenv func(string) string, userConfigDir func() (string, error)) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "gomodoro: config: expected a subcommand: path, init, or check")
		return exitUsageError
	}

	sub := args[0]
	configFlag, rest, err := extractConfigFlag(args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		return exitUsageError
	}
	if len(rest) > 0 {
		fmt.Fprintf(stderr, "gomodoro: config %s: unexpected argument(s): %s\n", sub, strings.Join(rest, " "))
		return exitUsageError
	}

	loc, err := config.ResolveLocation(configFlag, getenv, userConfigDir)
	if err != nil {
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		return exitRuntimeErr
	}

	switch sub {
	case "path":
		fmt.Fprintln(stdout, loc.Path)
		return exitOK

	case "init":
		if _, statErr := os.Stat(loc.Path); statErr == nil {
			fmt.Fprintf(stderr, "gomodoro: %s already exists; not overwriting\n", loc.Path)
			return exitRuntimeErr
		}
		if err := config.WriteDefaultFile(loc.Path); err != nil {
			fmt.Fprintf(stderr, "gomodoro: %v\n", err)
			return exitRuntimeErr
		}
		fmt.Fprintf(stdout, "Created %s\n", loc.Path)
		return exitOK

	case "check":
		if _, statErr := os.Stat(loc.Path); statErr != nil {
			if os.IsNotExist(statErr) && !loc.Explicit {
				fmt.Fprintf(stdout, "%s: not found; defaults will be used\n", loc.Path)
				return exitOK
			}
		}
		if _, err := config.Load(loc); err != nil {
			fmt.Fprintf(stderr, "gomodoro: %v\n", err)
			return exitRuntimeErr
		}
		fmt.Fprintf(stdout, "%s: OK\n", loc.Path)
		return exitOK

	default:
		fmt.Fprintf(stderr, "gomodoro: config: unknown subcommand %q (want path, init, or check)\n", sub)
		return exitUsageError
	}
}
