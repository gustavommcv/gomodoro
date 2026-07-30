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

Between phases, Gomodoro shows a progress bar, sends a desktop notification
(with sound, where the OS supports it), and waits for you to press `Enter`
before moving on to the next phase.

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

## Flags

| Flag       | Description                     | Default |
| ---------- | -------------------------------- | ------- |
| `-work`    | Work session duration (minutes, must be > 0) | 25      |
| `-sbreak`  | Short break duration (minutes, must be > 0)  | 5       |
| `-lbreak`  | Long break duration (minutes, must be > 0)   | 15      |

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
work but have not been verified in this repository yet.

## Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- No pause, resume, or skip while a phase is running.
- No configuration file or environment variable overrides yet; durations are
  set via flags for each run.
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
