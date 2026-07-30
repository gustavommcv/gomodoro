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
)

const (
	exitOK          = 0
	exitUsageError  = 2
	exitRuntimeErr  = 1
	exitInterrupted = 130 // conventional 128+SIGINT
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}

// run parses arguments, wires Ctrl+C cancellation, and executes one
// Pomodoro lap. It contains no os.Exit call so it can be exercised directly
// from tests with in-memory buffers; main is the only place that exits the
// process.
func run(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
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

	sleeper := pomodoro.NewDefaultSleeper(stdout, stderr, stdin)

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
