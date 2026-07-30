# Gomodoro
<img src="https://github.com/user-attachments/assets/9bdb38f8-423e-48c5-af2a-c5295197b644" width="200" alt="gomodoro-logo">

**A lightweight CLI Pomodoro timer for developers, built in Go.**

## Project status

Gomodoro is usable today for its core purpose: it runs one Pomodoro lap in
your terminal and gets out of your way. It has no config file, no themes,
and no continuous/looping mode yet — see [Roadmap](#-roadmap).

## What it does

Running `gomodoro` executes **one lap**: four work sessions, with a short
break after each of the first three and a long break after the fourth, then
the program exits.

```
Work -> Short break -> Work -> Short break -> Work -> Short break -> Work -> Long break
```

Between phases, Gomodoro shows a progress bar, then alerts you that the
phase ended — a desktop notification and a short alert sound, each
independently controllable — and waits for you to press `Enter` before
moving on to the next phase.

## Usage

Both forms below are equivalent — use whichever you prefer:

```bash
# Direct flags
gomodoro
gomodoro -work=25 -sbreak=5 -lbreak=15

# Explicit `start` subcommand
gomodoro start
gomodoro start -work=25 -sbreak=5 -lbreak=15
```

```bash
# Custom durations
gomodoro -work=50 -sbreak=10
gomodoro start -work=25 -sbreak=5 -lbreak=20
```

Press `Ctrl+C` at any time to stop; Gomodoro exits without a stack trace and
does not wait for the current tick or notification to finish.

Unknown flags and unexpected extra arguments are rejected with a clear
message on stderr and a non-zero exit code — they are never silently
ignored.

### Alerts

By default, both the desktop notification and the alert sound are on. Each
can be turned off independently — turning one off does not affect the
other:

```bash
# No visual notification, but the sound still plays
gomodoro start -notify=false

# No sound, but the notification still shows
gomodoro start -sound=false

# Neither alert; the rest of the flow (progress bar, waiting for Enter)
# is unchanged
gomodoro start -notify=false -sound=false

# Explicitly turn both on (same as the default)
gomodoro start -notify=true -sound=true
```

If a notification or sound fails to fire (e.g. no notification daemon or no
audio device available), Gomodoro prints a short warning to stderr and
continues — a failed alert never cancels the Pomodoro, and never affects
the other alert.

On Linux without a graphical session or configured audio, one or both
alerts may simply be unavailable; this is a platform/environment
limitation, not something Gomodoro can detect or work around in advance.

`-notify` and `-sound` currently only take effect for the current run. A
config file is planned (see [Roadmap](#-roadmap)); once it exists, the
intended precedence is `explicit flag > config file value > application
default` — flags will keep overriding the config file, not the other way
around.

## Flags

| Flag       | Description                     | Default |
| ---------- | -------------------------------- | ------- |
| `-work`    | Work session duration (minutes, must be > 0) | 25      |
| `-sbreak`  | Short break duration (minutes, must be > 0)  | 5       |
| `-lbreak`  | Long break duration (minutes, must be > 0)   | 15      |
| `-notify`  | Show a desktop notification at the end of each phase | true |
| `-sound`   | Play a short alert sound at the end of each phase | true |

`--notify=false` / `--sound=false` (double dash) also work — the standard
library's flag parser accepts either form.

Run `gomodoro -h` for the same reference from the terminal.

## Installation

### Using `go install`

```bash
go install github.com/gustavommcv/gomodoro/src/cmd/gomodoro@latest
```

This installs the `gomodoro` binary into your `$GOBIN` (or `$GOPATH/bin`).

### Manual build

```bash
git clone https://github.com/gustavommcv/gomodoro.git
cd gomodoro
go build -o gomodoro ./src/cmd/gomodoro
```

## Building and testing from source

```bash
go build ./...   # compile everything
go vet ./...      # static checks
go test ./...     # unit tests (fast, no real waiting — timing is simulated)
```

## Platform support

Gomodoro is written entirely against the Go standard library plus one
notification dependency ([`gen2brain/beeep`](https://github.com/gen2brain/beeep)),
both of which support Windows, Linux, and macOS. Development and manual
testing so far have only been done on **Windows**; Linux and macOS should
work but have not been verified in this repository yet. In particular, the
`-sound` alert on Linux depends on either PC-speaker access
(`/dev/input/by-path/platform-pcspkr-event-spkr`) or a terminal bell
fallback, and `-notify` depends on a running notification service (D-Bus,
`notify-send`, or `kdialog`) — headless or minimal environments may not
have either, and Gomodoro has not been exercised against one yet.

## Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- No pause, resume, or skip while a phase is running.
- No configuration file or environment variable overrides yet; durations and
  alert toggles are set via flags for each run.
- No session history or statistics.
- No color themes.

## 📌 Roadmap

- [ ] Config file support
- [ ] Continuous/looping execution (e.g. `--loop`)
- [ ] Pause / resume / skip
- [ ] Customizable terminal color themes
- [ ] Lap count / session history

## License

GPLv3 — see [LICENSE](LICENSE).
