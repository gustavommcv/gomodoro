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
moving on to the next phase. While a phase is running, and you're in a
real terminal, you can pause, resume, or skip it without touching Enter.

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

### Interactive controls

While a phase's countdown is running, and both stdin and stdout are real
terminals, Gomodoro shows and responds to:

```
Work  12:43 remaining  [████████░░░░░░░░] 48%  [p] Pause  [s] Skip  [Ctrl+C] Quit
```

| Key | Action |
| --- | --- |
| `p` / `P` | Pause, or resume if already paused (one key toggles both directions) |
| `s` / `S` | Skip the current phase |
| `Ctrl+C` | Quit immediately |

`p` is a single toggle, not separate pause/resume keys — this matches how
most terminal Pomodoro/timer tools do it (see this project's audio/UX
research history for the comparison), and it's one key to remember instead
of two. Any other key is simply ignored; the controls shown are the
complete, discoverable list, nothing else does anything.

**While paused**, the display adds a `PAUSED` marker and swaps the hint to
`[p] Resume`; the remaining time is frozen exactly where it was (no time
elapses while paused, and resuming doesn't lose or add time, however long
the pause lasted) and nothing else in the terminal changes or flickers.

**Skipping** a phase ends it immediately: it does *not* count as a normal
completion, so it does *not* trigger the end-of-phase notification or
sound, and — unlike a natural completion — it does *not* wait for you to
press Enter before moving on, since skipping is already a deliberate
keypress. You'll see a short "`<phase> skipped.`" line instead. Cancelling
with `Ctrl+C` is a third, distinct outcome from either: no alert, no
skip message, and the whole program exits (it doesn't move on to the next
phase).

**Without a real terminal on both ends** (input/output redirected to a
file, piped, or run non-interactively — e.g. `gomodoro start < /dev/null`
or `gomodoro start | tee log`), Gomodoro detects this and simply doesn't
offer these controls for that run: the countdown displays and behaves
exactly as it did before this feature existed, and Gomodoro never blocks
trying to read a keypress that can't arrive. `Ctrl+C` still works in this
mode too (via the normal OS signal, same as always).

Terminal raw mode (needed to read `p`/`s` without waiting for Enter) is
entered only around the countdown itself and always restored right after
— on natural completion, on skip, on cancellation, and on error — through
a single, idempotent restore step. It's implemented with
[`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term), the Go
project's own small terminal package (`MakeRaw`/`Restore`/`IsTerminal`;
no cgo, no extra runtime dependency) — not a general keyboard-event
library or a TUI framework, which this feature doesn't need.

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

Gomodoro's own code and its three dependencies
([`gen2brain/beeep`](https://github.com/gen2brain/beeep) for notifications,
[`ebitengine/oto`](https://github.com/ebitengine/oto) for sound,
[`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) for raw-mode
input) build without cgo on Windows, Linux, and macOS, for both `amd64` and
`arm64`. Being honest about what that does and doesn't prove:

| Platform | Compiled (cross-compilation) | Automated tests | Manually run | Notes |
| --- | --- | --- | --- | --- |
| `windows/amd64` | ✅ | ✅ | ✅ (real run: alerts fire, non-interactive fallback confirmed) | Primary development platform |
| `windows/arm64` | ✅ | — | — | Compiles; not run |
| `darwin/amd64` | ✅ | — | — | Compiles; not run |
| `darwin/arm64` | ✅ | — | — | Compiles; not run |
| `linux/amd64` | ✅ | — | — | Compiles; not run |
| `linux/arm64` | ✅ | — | — | Compiles; not run |

"Compiled" here means `GOOS`/`GOARCH` cross-compilation succeeded with
`CGO_ENABLED=0` from a Windows machine — it proves the code builds, not
that it runs correctly on that OS/CPU combination. Only `windows/amd64` has
actually been run manually, and even there, the development environment
used to build this feature has no real interactive terminal attached
(`term.IsTerminal` reports false), so the *interactive* pause/resume/skip
controls specifically have only been exercised by the automated test suite
(which drives the countdown's logic directly, without a real terminal or
raw mode) — not observed working end-to-end with a real keypress on any
platform yet. Linux and macOS runtime behavior — including the
PipeWire/PulseAudio/ALSA sound path, the notification daemon requirement,
and raw-mode behavior in a real terminal — is based on reading the
dependencies' own source and documentation, not on running Gomodoro there.
If you're on Linux (especially Hyprland/PipeWire), the
[validation steps](#validating-on-linuxhyprland) below are exactly how to
close that gap.

## Validating on Linux/Hyprland

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

# 6. Interactive controls -- start a longer phase so there's time to try
#    these, and confirm:
#    - pressing `p` freezes the countdown at exactly the remaining time
#      shown and adds a PAUSED marker, with no CPU spinning
#      (check e.g. `top`/`htop` -- gomodoro should sit near 0% CPU while
#      paused, not busy-loop);
#    - pressing `p` again resumes from exactly where it froze, not from
#      the start, and not with missing/extra time;
#    - pressing `s` (either while running or while paused) ends the phase
#      immediately with a short "skipped." message, no notification, no
#      sound, and moves straight to the next phase without waiting for
#      Enter:
./gomodoro start -work=5

# 7. Ctrl+C while a phase is actively running (i.e. while the interactive
#    controls are shown) -- should print "gomodoro: interrupted,
#    stopping." and exit immediately, with no alert firing afterward:
./gomodoro start -work=5
# (press Ctrl+C a few seconds in, while it's running, not paused)

# 8. Ctrl+C while a phase is paused -- should behave the same as step 7:
./gomodoro start -work=5
# (press `p` to pause, then Ctrl+C)

# 9. After every exit above (natural completion, skip, Ctrl+C), confirm
#    your shell's normal typing/echo still works right away -- type
#    something and see it appear normally. This is what proves the
#    terminal was actually restored out of raw mode, not just that the
#    process exited.
echo "terminal still behaves normally after gomodoro exits"

# 10. Confirm nothing was left behind:
pgrep -a gomodoro   # should be empty
# (Gomodoro does not open any persistent audio device/stream, and doesn't
# hold the terminal in raw mode after it exits -- each run cleans up after
# itself)

# 11. Non-interactive fallback -- confirm this does NOT show the
#     interactive hint text and does NOT hang waiting for a key that can
#     never arrive:
./gomodoro start -work=1 < /dev/null
./gomodoro start -work=1 | tee /tmp/gomodoro-log.txt
```

Please report back:
- whether step 2 was actually audible, and through which backend
  (`pactl`/`pw-cli` output helps identify that);
- whether pause/resume in step 6 genuinely froze and resumed without
  losing or gaining time;
- whether your shell's input behaves normally after each exit path in
  step 9 (this is the real test of whether raw mode was cleanly restored).

These are the pieces of this feature that could not be confirmed from this
environment (no real terminal, no real audio device attached here).

## Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- Pause/resume/skip only work in a real terminal (both stdin and stdout);
  redirected or piped runs fall back to the plain countdown with no
  interactive controls, by design (see [Interactive controls](#interactive-controls)).
- No configuration file or environment variable overrides yet; durations and
  alert toggles are set via flags for each run.
- No session history or statistics — a skipped phase is deliberately *not*
  counted the same as a completed one internally, so this distinction is
  preserved for whenever history/stats are added, but nothing is persisted
  or reported yet.
- No color themes.
- The sound backend ([`ebitengine/oto`](https://github.com/ebitengine/oto))
  is pinned to a pre-1.0 `v3.5.0-alpha` release, chosen deliberately over
  the current stable `v3.4.0` because the stable release requires cgo and
  ALSA development headers to build on Linux, while the alpha builds
  without cgo everywhere (see the audio backend research in this project's
  history for the full comparison). This is a real, tracked trade-off: an
  alpha dependency has no compatibility guarantee from its maintainers.
  Revisit this once `oto` ships a stable cgo-free release.
- The interactive controls have not yet been exercised with a real
  keypress on any platform in this project's own testing (see
  [Platform support](#platform-support)) — only through the automated
  test suite, which drives the same logic without a real terminal.

## 📌 Roadmap

- [ ] Config file support
- [ ] Continuous/looping execution (e.g. `--loop`)
- [ ] Customizable terminal color themes
- [ ] Lap count / session history

## License

GPLv3 — see [LICENSE](LICENSE).
