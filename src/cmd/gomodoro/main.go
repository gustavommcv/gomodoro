package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin, interactive))
}

// run parses arguments, wires Ctrl+C cancellation, and executes one
// Pomodoro lap. It contains no os.Exit call so it can be exercised directly
// from tests with in-memory buffers; main is the only place that exits the
// process.
func run(args []string, stdout, stderr io.Writer, stdin io.Reader, interactive bool) int {
	opts, err := parseArgs(args)
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
