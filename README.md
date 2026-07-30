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
the other alert. The sound alert never falls back to the terminal bell
(`\a`) — if it can't reach a real audio device, it warns honestly instead
of pretending to succeed.

`-notify` and `-sound` currently only take effect for the current run. A
config file is planned (see [Roadmap](#-roadmap)); once it exists, the
intended precedence is `explicit flag > config file value > application
default` — flags will keep overriding the config file, not the other way
around.

#### How the sound alert works

The alert sound is a short tone (~350ms, a single gentle pitch with a
fade-in and fade-out so there's no click at either end) generated in memory
at runtime — there's no embedded audio file and nothing is downloaded.
It's played through the operating system's normal audio output via
[`ebitengine/oto`](https://github.com/ebitengine/oto), **not** the PC
speaker (`pcspkr`) and **not** the terminal bell:

- **Windows** — WASAPI, no extra runtime requirements.
- **macOS** — the system AudioQueue API, no extra runtime requirements.
- **Linux** — PulseAudio's native protocol first (which is exactly what
  PipeWire serves via its `pipewire-pulse` compatibility layer, so this
  reaches PipeWire directly on a typical modern Linux setup, Hyprland
  included), falling back to ALSA if no Pulse-compatible socket is found.
  If neither is reachable, Gomodoro reports a warning — it does not fall
  back to `pcspkr` or `\a`.

None of this requires a C compiler or cgo to build on any of the three
platforms — see [Platform support](#platform-support) below for exactly
which combinations were compiled, tested, and run manually.

#### How the visual notification works

The notification uses [`gen2brain/beeep`](https://github.com/gen2brain/beeep)
(already a dependency for the tray/toast side of things): native toast or
balloon notifications on Windows, the system AudioQueue-adjacent
notification center via `osascript` on macOS, and on Linux, D-Bus (falling
back to the `notify-send` or `kdialog` commands if present) — which means
Linux needs a running notification daemon such as **Mako**, **SwayNC**, or
**Dunst** for `-notify` to actually show anything. The absence of such a
daemon does not block the timer or the sound alert: `-notify` and `-sound`
are completely independent, and the notification backend is never used to
trigger the sound (some notification daemons ignore sound hints entirely,
so the sound alert always plays through the audio backend directly,
regardless of what the notification daemon does or doesn't support).

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
go build ./...   # compile everything -- no C compiler or cgo needed on any of the three OSes
go vet ./...      # static checks
go test ./...     # unit tests (fast, no real waiting, no real sound/notifications fired)
```

## Platform support

Gomodoro's own code and its two dependencies
([`gen2brain/beeep`](https://github.com/gen2brain/beeep) for notifications,
[`ebitengine/oto`](https://github.com/ebitengine/oto) for sound) build
without cgo on Windows, Linux, and macOS, for both `amd64` and `arm64`.
Being honest about what that does and doesn't prove:

| Platform | Compiled (cross-compilation) | Automated tests | Manually run | Notes |
| --- | --- | --- | --- | --- |
| `windows/amd64` | ✅ | ✅ | ✅ (real run, both alerts fired without error) | Primary development platform |
| `windows/arm64` | ✅ | — | — | Compiles; not run |
| `darwin/amd64` | ✅ | — | — | Compiles; not run |
| `darwin/arm64` | ✅ | — | — | Compiles; not run |
| `linux/amd64` | ✅ | — | — | Compiles; not run |
| `linux/arm64` | ✅ | — | — | Compiles; not run |

"Compiled" here means `GOOS`/`GOARCH` cross-compilation succeeded with
`CGO_ENABLED=0` from a Windows machine — it proves the code builds, not
that it runs correctly on that OS/CPU combination. Only `windows/amd64` has
actually been run and observed to complete a real phase with both alerts
firing without error. Linux and macOS runtime behavior — including the
PipeWire/PulseAudio/ALSA path on Linux and the notification daemon
requirement described above — is based on reading the dependencies' own
source and documentation, not on running Gomodoro there yet. If you're on
Linux (especially Hyprland/PipeWire), the [validation steps](#validating-the-alerts-on-linuxhyprland)
below are exactly how to close that gap.

## Validating the alerts on Linux/Hyprland

```bash
# 0. Confirm the terminal bell is not what you're listening for --
#    this should stay silent or purely visual, never audible:
printf '\a'

# 1. Identify what's actually serving audio (informational only, this
#    doesn't change anything):
pactl info 2>/dev/null || pw-cli info 0 2>/dev/null
systemctl --user status pipewire pipewire-pulse 2>/dev/null

go build -o gomodoro ./src/cmd/gomodoro

# 2. Both alerts on (default) -- should show a notification (if a daemon
#    like Mako/SwayNC/Dunst is running) AND play an audible tone through
#    your normal output device:
./gomodoro start -work=1

# 3. Sound with notification explicitly disabled -- still audible:
./gomodoro start -work=1 -notify=false

# 4. Notification with sound explicitly disabled -- completely silent,
#    notification (if any) still shows:
./gomodoro start -work=1 -sound=false

# 5. If you can, temporarily stop your notification daemon (e.g.
#    `pkill mako`) and confirm the sound still plays and the timer still
#    completes -- the missing daemon should not block either.

# 6. Ctrl+C partway through a phase -- should print
#    "gomodoro: interrupted, stopping." and exit immediately, with no
#    alert firing afterward:
./gomodoro start -work=5
# (press Ctrl+C a few seconds in)

# 7. Confirm nothing was left behind:
pgrep -a gomodoro   # should be empty
# (Gomodoro does not open any persistent audio device/stream -- each
# alert is a single short-lived playback that finishes and returns)
```

Please report back whether step 2 was actually audible, and through which
backend (`pactl`/`pw-cli` output helps identify that) — that's the one
piece of this feature that could not be confirmed from this environment.

## Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- No pause, resume, or skip while a phase is running.
- No configuration file or environment variable overrides yet; durations and
  alert toggles are set via flags for each run.
- No session history or statistics.
- No color themes.
- The sound backend ([`ebitengine/oto`](https://github.com/ebitengine/oto))
  is pinned to a pre-1.0 `v3.5.0-alpha` release, chosen deliberately over
  the current stable `v3.4.0` because the stable release requires cgo and
  ALSA development headers to build on Linux, while the alpha builds
  without cgo everywhere (see the audio backend research in this project's
  history for the full comparison). This is a real, tracked trade-off: an
  alpha dependency has no compatibility guarantee from its maintainers.
  Revisit this once `oto` ships a stable cgo-free release.

## 📌 Roadmap

- [ ] Config file support
- [ ] Continuous/looping execution (e.g. `--loop`)
- [ ] Pause / resume / skip
- [ ] Customizable terminal color themes
- [ ] Lap count / session history

## License

GPLv3 — see [LICENSE](LICENSE).
