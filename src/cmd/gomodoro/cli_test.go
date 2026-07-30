package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Run("no arguments uses the documented defaults, including notify=true and sound=true", func(t *testing.T) {
		opts, err := parseArgs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("direct flags without a subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"-work=10", "-sbreak=2", "-lbreak=3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("start subcommand alone uses the documented defaults", func(t *testing.T) {
		opts, err := parseArgs([]string{"start"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("flags after the start subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-work=10", "-sbreak=2", "-lbreak=3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3, notify: true, sound: true}
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

	t.Run("-notify=false disables notification but leaves sound enabled", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.notify {
			t.Error("want notify=false")
		}
		if !opts.sound {
			t.Error("want sound to remain true; disabling notify must not disable sound")
		}
	})

	t.Run("-notify=true is accepted explicitly", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=true"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.notify {
			t.Error("want notify=true")
		}
	})

	t.Run("-sound=false disables sound but leaves notification enabled", func(t *testing.T) {
		opts, err := parseArgs([]string{"-sound=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.sound {
			t.Error("want sound=false")
		}
		if !opts.notify {
			t.Error("want notify to remain true; disabling sound must not disable notify")
		}
	})

	t.Run("-sound=true is accepted explicitly", func(t *testing.T) {
		opts, err := parseArgs([]string{"-sound=true"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.sound {
			t.Error("want sound=true")
		}
	})

	t.Run("both alerts disabled together", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=false", "-sound=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.notify || opts.sound {
			t.Errorf("want both disabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("both alerts explicitly enabled together", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=true", "-sound=true"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.notify || !opts.sound {
			t.Errorf("want both enabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("alert flags work after the start subcommand too", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-sound=false", "-notify=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.notify || opts.sound {
			t.Errorf("want both disabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("an invalid boolean value is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-notify=maybe"})
		if err == nil {
			t.Fatal("expected an error for an invalid boolean value")
		}
	})

	t.Run("help lists both alert flags", func(t *testing.T) {
		var buf bytes.Buffer
		printUsage(&buf)
		out := buf.String()
		if !strings.Contains(out, "-notify") {
			t.Error("expected usage text to document -notify")
		}
		if !strings.Contains(out, "-sound") {
			t.Error("expected usage text to document -sound")
		}
	})
}

func TestOptionalBool(t *testing.T) {
	t.Run("an unset flag resolves to the given default", func(t *testing.T) {
		var b optionalBool
		if got := b.resolve(true); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := b.resolve(false); got != false {
			t.Errorf("got %v, want false", got)
		}
	})

	t.Run("an explicitly-set flag overrides the default in either direction", func(t *testing.T) {
		var t1 optionalBool
		if err := t1.Set("false"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := t1.resolve(true); got != false {
			t.Errorf("explicit false must override default true, got %v", got)
		}

		var t2 optionalBool
		if err := t2.Set("true"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := t2.resolve(false); got != true {
			t.Errorf("explicit true must override default false, got %v", got)
		}
	})

	t.Run("Set rejects a non-boolean value", func(t *testing.T) {
		var b optionalBool
		if err := b.Set("maybe"); err == nil {
			t.Fatal("expected an error for a non-boolean value")
		}
	})

	t.Run("is a bool-shaped flag for the flag package's -flag (no =value) form", func(t *testing.T) {
		var b optionalBool
		if !b.IsBoolFlag() {
			t.Error("want IsBoolFlag() to be true")
		}
	})
}

func TestResolveOptions(t *testing.T) {
	// This is the seam a future config file slots into: resolveOptions is
	// the one place that decides precedence, so parseArgs itself never
	// needs to change when a config layer is added between "explicit flag"
	// and "application default".
	t.Run("explicit CLI flags take precedence over defaults", func(t *testing.T) {
		var notify, sound optionalBool
		if err := notify.Set("false"); err != nil {
			t.Fatal(err)
		}
		if err := sound.Set("true"); err != nil {
			t.Fatal(err)
		}

		got := resolveOptions(25, 5, 15, notify, sound)

		if got.notify != false {
			t.Errorf("want notify=false (explicit), got %v", got.notify)
		}
		if got.sound != true {
			t.Errorf("want sound=true (explicit), got %v", got.sound)
		}
	})

	t.Run("unset flags fall back to the application defaults", func(t *testing.T) {
		var notify, sound optionalBool

		got := resolveOptions(25, 5, 15, notify, sound)

		if got.notify != true {
			t.Errorf("want notify to fall back to the default (true), got %v", got.notify)
		}
		if got.sound != true {
			t.Errorf("want sound to fall back to the default (true), got %v", got.sound)
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
