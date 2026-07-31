package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/gustavommcv/gomodoro/src/internal/config"
	"github.com/gustavommcv/gomodoro/src/internal/pomodoro"
	"github.com/gustavommcv/gomodoro/src/pkg/rawterm"
)

const (
	exitOK          = 0
	exitUsageError  = 2
	exitRuntimeErr  = 1
	exitInterrupted = 130 // conventional 128+SIGINT
)

func main() {
	// Interactive controls (pause/resume/skip) only make sense with a real
	// terminal on both ends; this is decided here, once, from the real
	// os.Stdin/os.Stdout, since run() itself is tested with in-memory
	// buffers that have no meaningful "is this a terminal" answer.
	interactive := rawterm.Interactive()
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin, interactive, os.Getenv, os.UserConfigDir))
}

// run parses arguments, resolves and loads the configuration, wires
// Ctrl+C cancellation, and executes one Pomodoro lap. It contains no
// os.Exit call so it can be exercised directly from tests with in-memory
// buffers; main is the only place that exits the process. getenv and
// userConfigDir are injected (os.Getenv/os.UserConfigDir in real use) so
// config resolution is testable without touching the real environment.
func run(args []string, stdout, stderr io.Writer, stdin io.Reader, interactive bool, getenv func(string) string, userConfigDir func() (string, error)) int {
	// help and --version must work even with a broken config file, so
	// they're checked before any config is loaded.
	if len(args) > 0 {
		switch args[0] {
		case "help":
			printUsage(stdout)
			return exitOK
		case "--version", "-version":
			printVersion(stdout)
			return exitOK
		case "config":
			return runConfigCommand(args[1:], stdout, stderr, getenv, userConfigDir)
		}
	}
	if hasHelpFlag(args) {
		printUsage(stdout)
		return exitOK
	}

	configFlag, rest, err := extractConfigFlag(args)
	if err != nil {
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		printUsage(stderr)
		return exitUsageError
	}

	loc, err := config.ResolveLocation(configFlag, getenv, userConfigDir)
	if err != nil {
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		return exitRuntimeErr
	}

	// A config error here (a broken or unreadable file) must stop before
	// anything else starts: no timer, no alerts, no raw mode. Absence of
	// the file at the default location is not an error -- Load already
	// handles that by returning Defaults().
	base, err := config.Load(loc)
	if err != nil {
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		return exitRuntimeErr
	}

	opts, err := parseArgs(rest, base)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return exitOK
		}
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		printUsage(stderr)
		return exitUsageError
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	sleeper := pomodoro.NewDefaultSleeper(stdout, stderr, stdin, opts.notify, opts.sound, interactive, rawterm.EnterRaw)

	err = pomodoro.Pomodoro(ctx, sleeper, opts.work, opts.sbreak, opts.lbreak)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(stderr, "\ngomodoro: interrupted, stopping.")
			return exitInterrupted
		}
		fmt.Fprintf(stderr, "gomodoro: %v\n", err)
		return exitRuntimeErr
	}

	return exitOK
}
