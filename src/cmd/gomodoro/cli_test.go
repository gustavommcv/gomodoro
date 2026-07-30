package main

import (
	"bytes"
	"errors"
	"flag"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Run("no arguments uses the documented defaults", func(t *testing.T) {
		opts, err := parseArgs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("direct flags without a subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"-work=10", "-sbreak=2", "-lbreak=3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("start subcommand alone uses the documented defaults", func(t *testing.T) {
		opts, err := parseArgs([]string{"start"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("flags after the start subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-work=10", "-sbreak=2", "-lbreak=3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("the old and new syntaxes are equivalent", func(t *testing.T) {
		direct, err := parseArgs([]string{"-work=42"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		started, err := parseArgs([]string{"start", "-work=42"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if direct != started {
			t.Errorf("expected equivalent results, got %+v vs %+v", direct, started)
		}
	})

	t.Run("a zero duration is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-work=0"})
		if err == nil {
			t.Fatal("expected an error for a zero work duration")
		}
	})

	t.Run("a negative duration is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-sbreak=-5"})
		if err == nil {
			t.Fatal("expected an error for a negative short break duration")
		}
	})

	t.Run("an unknown flag is rejected, not ignored", func(t *testing.T) {
		_, err := parseArgs([]string{"-bogus=1"})
		if err == nil {
			t.Fatal("expected an error for an unknown flag")
		}
	})

	t.Run("an unexpected trailing positional argument is rejected, not ignored", func(t *testing.T) {
		_, err := parseArgs([]string{"-work=25", "extra"})
		if err == nil {
			t.Fatal("expected an error for an unexpected positional argument")
		}
	})

	t.Run("this used to be the silently-ignored bug: start followed by flags that never got parsed", func(t *testing.T) {
		// Historically, `gomodoro start --work 50` fell straight through to
		// defaults because flag.Parse stopped at the first non-flag token
		// ("start") and nothing ever consumed it. Confirm -work=50 really
		// takes effect now instead of being silently dropped.
		opts, err := parseArgs([]string{"start", "-work=50"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.work != 50 {
			t.Errorf("expected -work=50 after start to take effect, got work=%d", opts.work)
		}
	})

	t.Run("an unrecognized leading token is rejected, not silently treated as a no-op", func(t *testing.T) {
		_, err := parseArgs([]string{"bogus"})
		if err == nil {
			t.Fatal("expected an error for an unrecognized leading argument")
		}
	})

	t.Run("help is reported distinctly from a usage error", func(t *testing.T) {
		_, err := parseArgs([]string{"-h"})
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("an unknown flag exits non-zero and writes a message to stderr", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-bogus"}, &stdout, &stderr, nil)

		if code == exitOK {
			t.Fatalf("expected a non-zero exit code, got %d", code)
		}
		if stderr.Len() == 0 {
			t.Error("expected an error message on stderr")
		}
	})

	t.Run("help exits zero and writes usage to stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-h"}, &stdout, &stderr, nil)

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d", exitOK, code)
		}
		if stdout.Len() == 0 {
			t.Error("expected usage text on stdout")
		}
	})
}
