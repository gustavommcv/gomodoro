// Package rawterm connects Gomodoro's interactive phase controls to the
// real terminal: detecting whether stdin/stdout are usable terminals at
// all, and putting stdin into raw (uncooked) input mode so single
// keypresses can be read without waiting for Enter. It wraps
// golang.org/x/term, the Go team's own small, dependency-light terminal
// package, rather than a general keyboard-event library or a TUI
// framework -- Gomodoro only needs single-byte reads and mode toggling.
package rawterm

import (
	"os"
	"sync"

	"golang.org/x/term"
)

// Interactive reports whether both stdin and stdout are connected to a
// real terminal. Interactive phase controls (pause/resume/skip) should
// only be offered when this is true; a redirected or piped stdin/stdout
// falls back to the plain, non-interactive countdown, which needs neither
// raw mode nor single-key input.
func Interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// EnterRaw puts stdin into raw mode (no echo, no line buffering, no
// signal-generating control characters) and returns a function that
// restores the terminal to whatever state it was in before. restore is
// idempotent -- calling it more than once after the first call is a
// harmless no-op, so callers can safely defer it without worrying about a
// second explicit restore elsewhere also being safe.
func EnterRaw() (restore func(), err error) {
	fd := int(os.Stdin.Fd())

	old, err := term.MakeRaw(fd)
	if err != nil {
		return func() {}, err
	}

	var once sync.Once
	restore = func() {
		once.Do(func() {
			_ = term.Restore(fd, old)
		})
	}
	return restore, nil
}
