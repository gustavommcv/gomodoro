# Gomodoro
<img src="https://github.com/user-attachments/assets/9bdb38f8-423e-48c5-af2a-c5295197b644" width="200" alt="gomodoro-logo">

**A lightweight CLI Pomodoro timer for developers, built in Go.**

## Project status

Gomodoro is usable today for its core purpose: it runs one Pomodoro lap in
your terminal and gets out of your way. It has no themes and no
continuous/looping mode yet — see [Roadmap](#-roadmap).

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
moving on to the next phase. While a phase is running, and you're in a
real terminal, you can pause, resume, or skip it without touching Enter.

## Quick start

```bash
gomodoro
gomodoro -work=25 -sbreak=5 -lbreak=15
```

Press `Ctrl+C` at any time to stop. See [Usage](docs/usage.md) for the
interactive pause/resume/skip controls and how alerts work.

## Configuration

Gomodoro can be configured with a **Lua** file — inspired by Hyprland's own
`hyprland.lua` — that supports both a single file and `require()`-based
modules, runs in a tightly sandboxed interpreter (no filesystem, process,
or network access), and is looked up via `os.UserConfigDir()`:

```lua
return {
	version = 1,
	alerts = {
		sound = false,
	},
}
```

See [Configuration](docs/configuration.md) for the full schema, the
modular layout, the sandbox details, `--config`/`GOMODORO_CONFIG`,
precedence rules, and the `gomodoro config path|init|check` commands.

## Flags

| Flag        | Description                     | Default |
| ----------- | -------------------------------- | ------- |
| `-work`     | Work session duration (minutes, must be > 0) | 25      |
| `-sbreak`   | Short break duration (minutes, must be > 0)  | 5       |
| `-lbreak`   | Long break duration (minutes, must be > 0)   | 15      |
| `-notify`   | Show a desktop notification at the end of each phase | true |
| `-sound`    | Play a short alert sound at the end of each phase | true |
| `--config`  | Path to a config file (works before or after `start`) | resolved via [Locations](docs/configuration.md#locations) |

`--notify=false` / `--sound=false` (double dash) also work — the standard
library's flag parser accepts either form. Run `gomodoro -h` for the same
reference from the terminal.

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
go build ./...   # compile everything -- no C compiler or cgo needed on any of the three OSes
go vet ./...      # static checks
go test ./...     # unit tests (fast, no real waiting, no real sound/notifications fired)
```

## Documentation

| Doc | Covers |
| --- | --- |
| [Usage](docs/usage.md) | Flags, interactive controls, how alerts (sound/notification) work |
| [Configuration](docs/configuration.md) | The Lua config file, `require()`, sandbox, schema, locations, `config` commands |
| [Platform support](docs/platform-support.md) | What's compiled, automated-tested, and manually run, per OS/arch |
| [Validating on Linux/Hyprland](docs/validating-linux.md) | Manual verification script for platforms not yet run by hand |
| [Current limitations](docs/limitations.md) | Known gaps, tracked deliberately |

## 📌 Roadmap

- [ ] Continuous/looping execution (e.g. `--loop`)
- [ ] Customizable terminal color themes (via the reserved `style` section)
- [ ] `gomodoro config show` / `--effective` (print the fully-resolved config)
- [ ] Lap count / session history

## License

GPLv3 — see [LICENSE](LICENSE).
