package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gustavommcv/gomodoro/src/internal/config"
)

func TestParseArgs(t *testing.T) {
	defaults := config.Defaults()

	t.Run("no arguments uses the documented defaults, including notify=true and sound=true", func(t *testing.T) {
		opts, err := parseArgs(nil, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("direct flags without a subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"-work=10", "-sbreak=2", "-lbreak=3"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("start subcommand alone uses the documented defaults", func(t *testing.T) {
		opts, err := parseArgs([]string{"start"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 25, sbreak: 5, lbreak: 15, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("flags after the start subcommand", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-work=10", "-sbreak=2", "-lbreak=3"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := options{work: 10, sbreak: 2, lbreak: 3, notify: true, sound: true}
		if opts != want {
			t.Errorf("got %+v, want %+v", opts, want)
		}
	})

	t.Run("the old and new syntaxes are equivalent", func(t *testing.T) {
		direct, err := parseArgs([]string{"-work=42"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		started, err := parseArgs([]string{"start", "-work=42"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if direct != started {
			t.Errorf("expected equivalent results, got %+v vs %+v", direct, started)
		}
	})

	t.Run("a zero duration is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-work=0"}, defaults)
		if err == nil {
			t.Fatal("expected an error for a zero work duration")
		}
	})

	t.Run("a negative duration is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-sbreak=-5"}, defaults)
		if err == nil {
			t.Fatal("expected an error for a negative short break duration")
		}
	})

	t.Run("an unknown flag is rejected, not ignored", func(t *testing.T) {
		_, err := parseArgs([]string{"-bogus=1"}, defaults)
		if err == nil {
			t.Fatal("expected an error for an unknown flag")
		}
	})

	t.Run("an unexpected trailing positional argument is rejected, not ignored", func(t *testing.T) {
		_, err := parseArgs([]string{"-work=25", "extra"}, defaults)
		if err == nil {
			t.Fatal("expected an error for an unexpected positional argument")
		}
	})

	t.Run("this used to be the silently-ignored bug: start followed by flags that never got parsed", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-work=50"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.work != 50 {
			t.Errorf("expected -work=50 after start to take effect, got work=%d", opts.work)
		}
	})

	t.Run("an unrecognized leading token is rejected, not silently treated as a no-op", func(t *testing.T) {
		_, err := parseArgs([]string{"bogus"}, defaults)
		if err == nil {
			t.Fatal("expected an error for an unrecognized leading argument")
		}
	})

	t.Run("help is reported distinctly from a usage error", func(t *testing.T) {
		_, err := parseArgs([]string{"-h"}, defaults)
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})

	t.Run("-notify=false disables notification but leaves sound enabled", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=false"}, defaults)
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
		opts, err := parseArgs([]string{"-notify=true"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.notify {
			t.Error("want notify=true")
		}
	})

	t.Run("-sound=false disables sound but leaves notification enabled", func(t *testing.T) {
		opts, err := parseArgs([]string{"-sound=false"}, defaults)
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
		opts, err := parseArgs([]string{"-sound=true"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.sound {
			t.Error("want sound=true")
		}
	})

	t.Run("both alerts disabled together", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=false", "-sound=false"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.notify || opts.sound {
			t.Errorf("want both disabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("both alerts explicitly enabled together", func(t *testing.T) {
		opts, err := parseArgs([]string{"-notify=true", "-sound=true"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.notify || !opts.sound {
			t.Errorf("want both enabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("alert flags work after the start subcommand too", func(t *testing.T) {
		opts, err := parseArgs([]string{"start", "-sound=false", "-notify=false"}, defaults)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.notify || opts.sound {
			t.Errorf("want both disabled, got notify=%v sound=%v", opts.notify, opts.sound)
		}
	})

	t.Run("an invalid boolean value is rejected", func(t *testing.T) {
		_, err := parseArgs([]string{"-notify=maybe"}, defaults)
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

	t.Run("help lists the interactive phase controls", func(t *testing.T) {
		var buf bytes.Buffer
		printUsage(&buf)
		out := buf.String()
		for _, want := range []string{"Pause", "Skip", "Ctrl+C"} {
			if !strings.Contains(out, want) {
				t.Errorf("expected usage text to mention %q", want)
			}
		}
	})

	t.Run("help documents --config but lists no flag that doesn't exist", func(t *testing.T) {
		var buf bytes.Buffer
		printUsage(&buf)
		out := buf.String()
		if !strings.Contains(out, "--config") {
			t.Error("expected usage text to document --config")
		}
		if strings.Contains(out, "-theme") || strings.Contains(out, "-loop") {
			t.Error("usage text must not list flags that don't exist")
		}
	})

	t.Run("precedence: an unset flag falls back to the config-file-overlaid base, not the compiled default", func(t *testing.T) {
		base := config.Defaults()
		base.Alerts.Sound = false // as if a config file set alerts.sound = false
		base.Timer.Work = 50 * time.Minute

		opts, err := parseArgs(nil, base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.sound != false {
			t.Errorf("sound = %v, want false (from the config-file base)", opts.sound)
		}
		if opts.work != 50 {
			t.Errorf("work = %v, want 50 (from the config-file base)", opts.work)
		}
	})

	t.Run("precedence: an explicit flag overrides the config-file-overlaid base in either direction", func(t *testing.T) {
		base := config.Defaults()
		base.Alerts.Sound = false

		opts, err := parseArgs([]string{"-sound=true"}, base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.sound != true {
			t.Errorf("sound = %v, want true (explicit flag beats the config file)", opts.sound)
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

func TestOptionalInt(t *testing.T) {
	t.Run("an unset flag resolves to the given default", func(t *testing.T) {
		var o optionalInt
		if got := o.resolve(25); got != 25 {
			t.Errorf("got %d, want 25", got)
		}
	})

	t.Run("an explicitly-set flag overrides the default, even when it's zero", func(t *testing.T) {
		var o optionalInt
		if err := o.Set("0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := o.resolve(25); got != 0 {
			t.Errorf("explicit 0 must override the default, got %d", got)
		}
	})

	t.Run("Set rejects a non-integer value", func(t *testing.T) {
		var o optionalInt
		if err := o.Set("abc"); err == nil {
			t.Fatal("expected an error for a non-integer value")
		}
	})
}

func TestResolveOptions(t *testing.T) {
	base := config.Config{
		Timer:  config.TimerConfig{Work: 25 * time.Minute, ShortBreak: 5 * time.Minute, LongBreak: 15 * time.Minute},
		Alerts: config.AlertConfig{Notify: true, Sound: true},
	}

	t.Run("explicit CLI flags take precedence over the base", func(t *testing.T) {
		var notify, sound optionalBool
		notify.Set("false")
		sound.Set("true")
		var work, sbreak, lbreak optionalInt

		got := resolveOptions(work, sbreak, lbreak, notify, sound, base)

		if got.notify != false {
			t.Errorf("want notify=false (explicit), got %v", got.notify)
		}
		if got.sound != true {
			t.Errorf("want sound=true (explicit), got %v", got.sound)
		}
	})

	t.Run("unset flags fall back to the base (defaults or config-file-overlaid)", func(t *testing.T) {
		var work, sbreak, lbreak optionalInt
		var notify, sound optionalBool

		got := resolveOptions(work, sbreak, lbreak, notify, sound, base)

		if got.work != 25 || got.sbreak != 5 || got.lbreak != 15 {
			t.Errorf("got durations %+v, want 25/5/15 from base", got)
		}
		if got.notify != true || got.sound != true {
			t.Errorf("got alerts %+v, want true/true from base", got)
		}
	})
}

func TestExtractConfigFlag(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantVal  string
		wantRest []string
	}{
		{"absent", []string{"start", "-work=25"}, "", []string{"start", "-work=25"}},
		{"--config=value form", []string{"--config=/a/b.lua", "start"}, "/a/b.lua", []string{"start"}},
		{"-config=value form", []string{"-config=/a/b.lua", "start"}, "/a/b.lua", []string{"start"}},
		{"--config value (separate token)", []string{"start", "--config", "/a/b.lua"}, "/a/b.lua", []string{"start"}},
		{"before start", []string{"--config", "/a/b.lua", "start", "-work=10"}, "/a/b.lua", []string{"start", "-work=10"}},
		{"after start", []string{"start", "-work=10", "--config", "/a/b.lua"}, "/a/b.lua", []string{"start", "-work=10"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			val, rest, err := extractConfigFlag(c.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val != c.wantVal {
				t.Errorf("value = %q, want %q", val, c.wantVal)
			}
			if len(rest) != len(c.wantRest) {
				t.Fatalf("rest = %v, want %v", rest, c.wantRest)
			}
			for i := range rest {
				if rest[i] != c.wantRest[i] {
					t.Errorf("rest = %v, want %v", rest, c.wantRest)
				}
			}
		})
	}

	t.Run("a dangling --config with no value is an error, not silently ignored", func(t *testing.T) {
		_, _, err := extractConfigFlag([]string{"start", "--config"})
		if err == nil {
			t.Fatal("expected an error for --config with no value")
		}
	})
}

func TestRun(t *testing.T) {
	fakeUserConfigDir := func(dir string) func() (string, error) {
		return func() (string, error) { return dir, nil }
	}
	noEnv := func(string) string { return "" }

	t.Run("an unknown flag exits non-zero and writes a message to stderr", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-bogus"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(t.TempDir()))

		if code == exitOK {
			t.Fatalf("expected a non-zero exit code, got %d", code)
		}
		if stderr.Len() == 0 {
			t.Error("expected an error message on stderr")
		}
	})

	t.Run("-h exits zero and writes usage to stdout, without touching any config file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		// userConfigDir deliberately errors: if help touched config
		// resolution, this would surface as a failure.
		brokenDir := func() (string, error) { return "", errors.New("should not be called") }
		code := run([]string{"-h"}, &stdout, &stderr, nil, false, noEnv, brokenDir)

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d", exitOK, code)
		}
		if stdout.Len() == 0 {
			t.Error("expected usage text on stdout")
		}
	})

	t.Run("help (bare word) also exits zero with usage on stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"help"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(t.TempDir()))
		if code != exitOK {
			t.Fatalf("want exit code %d, got %d", exitOK, code)
		}
		if stdout.Len() == 0 {
			t.Error("expected usage text on stdout")
		}
	})

	t.Run("--version exits zero and prints a version, without touching any config file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		brokenDir := func() (string, error) { return "", errors.New("should not be called") }
		code := run([]string{"--version"}, &stdout, &stderr, nil, false, noEnv, brokenDir)

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d", exitOK, code)
		}
		if !strings.Contains(stdout.String(), "gomodoro") {
			t.Errorf("expected the version output to mention gomodoro, got: %q", stdout.String())
		}
	})

	t.Run("a broken default config file blocks the run with a clear error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "gomodoro"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "gomodoro", "config.lua"), []byte("return {\n version = ,\n}"), 0o644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := run(nil, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code == exitOK {
			t.Fatal("expected a non-zero exit code for a broken config file")
		}
		if stderr.Len() == 0 {
			t.Error("expected an error message on stderr")
		}
	})

	t.Run("but config check still works against that same broken file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "gomodoro"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "gomodoro", "config.lua"), []byte("return {\n version = ,\n}"), 0o644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "check"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code == exitOK {
			t.Fatal("expected config check to report the file as invalid")
		}
		if stderr.Len() == 0 {
			t.Error("expected the validation error on stderr")
		}
	})

	t.Run("a missing default config file is not an error: the run proceeds with defaults", func(t *testing.T) {
		// We can't actually run the timer to completion quickly, but we
		// can confirm parsing/config resolution didn't fail by checking
		// that the only way this would fail immediately is on an
		// unrelated flag error -- use -work=0 to force a fast, deterministic
		// exit through validate() instead of starting the real timer.
		var stdout, stderr bytes.Buffer
		code := run([]string{"-work=0"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(t.TempDir()))

		if code != exitUsageError {
			t.Fatalf("want exit code %d (usage error from -work=0), got %d -- if config resolution itself had failed, this would be exitRuntimeErr instead", exitUsageError, code)
		}
	})
}

func TestRunConfigCommand(t *testing.T) {
	fakeUserConfigDir := func(dir string) func() (string, error) {
		return func() (string, error) { return dir, nil }
	}
	noEnv := func(string) string { return "" }

	t.Run("config path prints only the resolved path to stdout", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "path"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d (stderr: %s)", exitOK, code, stderr.String())
		}
		want := filepath.Join(dir, "gomodoro", "config.lua") + "\n"
		if stdout.String() != want {
			t.Errorf("stdout = %q, want %q", stdout.String(), want)
		}
	})

	t.Run("config path does not create or read anything", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		run([]string{"config", "path"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if _, err := os.Stat(filepath.Join(dir, "gomodoro")); err == nil {
			t.Error("config path must not create the config directory")
		}
	})

	t.Run("config init creates the file with real defaults and reports the path", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "init"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d (stderr: %s)", exitOK, code, stderr.String())
		}
		path := filepath.Join(dir, "gomodoro", "config.lua")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected the config file to exist: %v", err)
		}
		if !strings.Contains(stdout.String(), path) {
			t.Errorf("expected stdout to mention the created path, got: %q", stdout.String())
		}
	})

	t.Run("config init refuses to overwrite an existing file", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, "gomodoro")
		os.MkdirAll(configDir, 0o755)
		existing := filepath.Join(configDir, "config.lua")
		os.WriteFile(existing, []byte("-- my custom config\nreturn {version = 1}"), 0o644)

		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "init"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code == exitOK {
			t.Fatal("expected config init to refuse to overwrite an existing file")
		}
		content, _ := os.ReadFile(existing)
		if !strings.Contains(string(content), "my custom config") {
			t.Error("the existing file must not have been touched")
		}
	})

	t.Run("config check reports success for a valid file", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, "gomodoro")
		os.MkdirAll(configDir, 0o755)
		os.WriteFile(filepath.Join(configDir, "config.lua"), []byte("return {version = 1}"), 0o644)

		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "check"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d (stderr: %s)", exitOK, code, stderr.String())
		}
	})

	t.Run("config check reports a missing default file as fine (defaults apply), not an error", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "check"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d (stderr: %s)", exitOK, code, stderr.String())
		}
	})

	t.Run("config check does not start the timer, alerts, or raw mode", func(t *testing.T) {
		// If it did, this call would hang or attempt real I/O; the test
		// completing quickly is itself the proof.
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		done := make(chan int, 1)
		go func() {
			done <- run([]string{"config", "check"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("config check did not return promptly")
		}
	})

	t.Run("--config selects an explicit file for any config subcommand", func(t *testing.T) {
		dir := t.TempDir()
		explicit := filepath.Join(dir, "custom.lua")
		os.WriteFile(explicit, []byte("return {version = 1}"), 0o644)

		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "path", "--config", explicit}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(dir))

		if code != exitOK {
			t.Fatalf("want exit code %d, got %d (stderr: %s)", exitOK, code, stderr.String())
		}
		if strings.TrimSpace(stdout.String()) != explicit {
			t.Errorf("stdout = %q, want %q", stdout.String(), explicit)
		}
	})

	t.Run("an unknown config subcommand is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"config", "bogus"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(t.TempDir()))
		if code != exitUsageError {
			t.Fatalf("want exit code %d, got %d", exitUsageError, code)
		}
	})

	t.Run("config with no subcommand is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"config"}, &stdout, &stderr, nil, false, noEnv, fakeUserConfigDir(t.TempDir()))
		if code != exitUsageError {
			t.Fatalf("want exit code %d, got %d", exitUsageError, code)
		}
	})
}
