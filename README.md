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

`-notify` and `-sound`, like the three durations, can also be set in the
[config file](#configuration) — an explicit flag always overrides the
config file, which always overrides the compiled-in default.

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

## Configuration

Gomodoro can be configured with a **Lua** file. The inspiration is
Hyprland, the Wayland compositor: it moved its own configuration to Lua in
v0.55 (May 2026; the current stable is v0.56.1), because plain
`key = value` files stop scaling once composition and reuse matter, while
Lua tables read almost as cleanly as a static file but let you factor
shared pieces out with `require()`. Gomodoro follows that same shape:

```lua
local timer = require("settings.timer")
local alerts = require("settings.alerts")
local style = require("themes.default")

return {
	version = 1,
	timer = timer,
	alerts = alerts,
	style = style,
}
```

**This is not "Lua" in the way an unrestricted embedded interpreter would
be.** Gomodoro's config Lua cannot touch the filesystem, run a process, use
the network, or execute anything outside the exact file(s) you point it
at. See [Sandbox](#sandbox) below for the precise, complete list of what is
and isn't available — not just "it's safe," but exactly which standard
library functions exist and which don't.

### A single file is enough

You don't need multiple files or directories. A minimal, complete
`config.lua` is:

```lua
return {
	version = 1,
	alerts = {
		sound = false,
	},
}
```

Every field you don't mention keeps its compiled-in default (see
[Defaults](#defaults) below) — this file changes only `alerts.sound`.

### Or split it into modules

```text
gomodoro/
├── config.lua
├── settings/
│   ├── timer.lua
│   └── alerts.lua
└── themes/
    └── default.lua
```

`config.lua`:

```lua
local timer = require("settings.timer")
local alerts = require("settings.alerts")
local style = require("themes.default")

return {
	version = 1,
	timer = timer,
	alerts = alerts,
	style = style,
}
```

`settings/timer.lua`:

```lua
return {
	work = "25m",
	short_break = "5m",
	long_break = "15m",
}
```

`settings/alerts.lua`:

```lua
return {
	notify = true,
	sound = true,
}
```

`themes/default.lua`:

```lua
return {
	theme = "default",
}
```

Nothing forces this exact layout — organize modules however you like, as
long as `require()` names point at real files (see below).

### `require()` resolution

`require("settings.timer")` resolves to `settings/timer.lua`, **relative
to the directory the main config file is in** — never the current working
directory, never `LUA_PATH`/`LUA_CPATH`. The rules:

- dots become path separators (`settings.timer` → `settings/timer.lua`),
  using Go's own portable path handling, so this works identically on
  Windows, Linux, and macOS;
- module names may only contain letters, digits, underscores, and dots as
  separators — no `/`, no `\`, no leading digit, no empty segment;
  absolute paths and `..` are rejected outright, by construction, not by a
  blocklist;
- each module is loaded and run **at most once** per config load, even if
  `require()`'d from multiple places, and the result is cached and reused;
- a `require()` cycle (`a` requires `b` requires `a`) is detected and
  reported with the full chain, instead of recursing forever;
- import depth and the total number of modules loaded are both capped
  (16 and 64 respectively) — small, documented limits meant to catch a
  mistake, not to constrain any reasonable config;
- a symlink inside the config directory that points outside it is
  rejected, the same as a `..` would be — there is no way for `require()`
  to read anything outside the directory the main config file lives in.

A load failure reports the full chain of files that were being loaded,
then the specific problem:

```text
failed to load Gomodoro configuration:
config.lua -> settings/timer.lua
settings/timer.lua:2: field "work" must be a valid positive duration
```

### Sandbox

The Lua state Gomodoro creates opens **only**: the `table`, `string`, and
`math` standard libraries (each function in those three does pure data
manipulation — none of them touch a file, a process, or the network), a
small hand-picked subset of the base library (`type`, `tostring`,
`tonumber`, `pairs`, `ipairs`, `next`, `select`, `assert`, `error`), and
Gomodoro's own `require()`.

**Never opened at all** — not opened-then-removed, simply never
registered in the first place: `os`, `io`, `debug`, and the standard
`package` library (which is what the stock `require`/`package.path` come
from; Gomodoro's `require()` is a separate, Go-implemented function, not
that one with some parts removed).

**Explicitly removed** after being added by the base library, because
gopher-lua (the embedded Lua runtime) doesn't offer a way to open only
part of it: `load`, `loadstring`, `loadfile`, `dofile` (load/run arbitrary
code), `module`, `newproxy` (module-system and proxy tricks),
`getfenv`/`setfenv` (Lua 5.1's environment-manipulation functions — a
known sandbox-escape vector), `print`, `collectgarbage` (not needed),
`pcall`/`xpcall` (error recovery isn't needed for a declarative config —
keeping failures fatal is the point), `rawget`/`rawset`/`rawequal`,
`unpack`, `setmetatable`/`getmetatable` (removes a whole class of exotic
table tricks), and the global `_G` table itself. `string.dump` is removed
individually too, though it's inert without `load`/`loadstring` present.
This exact list is enumerated directly from gopher-lua's own source and
checked by name in the test suite — nothing on it can silently reappear.

**Execution limits**, all real (see below, not "a goroutine we gave up
waiting on"):

- 64 KiB max per `.lua` file;
- 64 modules max per config load;
- 16 levels of `require()` nesting max;
- 500ms max total execution time for the whole load (main file plus every
  module it requires).

The time limit is enforced through the Lua runtime's own context-based
interruption (`SetContext` + a deadline): an infinite loop is actually
stopped mid-execution, not left running in an abandoned goroutine — this
was verified directly (an unbounded `while true do end` returns a real
error within the configured bound, every time). Once a config finishes
loading — successfully or not — the Lua runtime is closed; nothing from it
stays alive afterward.

### Defaults

```text
timer.work         25m
timer.short_break   5m
timer.long_break   15m
alerts.notify      true
alerts.sound       true
style.theme        "default"
```

These are the same values Gomodoro has always used — introducing the
config file did not change any default. There is exactly one place in the
source these come from; the CLI's flag defaults, the loader's overlay
base, and the file `gomodoro config init` generates are all guaranteed to
agree with it (checked by a test that loads the generated file back
through the real loader and compares it to the defaults, rather than by
copy-pasting the values in multiple places).

### Durations

Durations are plain `time.ParseDuration`-style strings (`"25m"`, `"1h"`,
`"90s"`), with one additional rule: **they must amount to a whole number
of minutes.** Gomodoro's timer itself only operates in whole minutes today
(that didn't change with this feature), so `"25m30s"` is rejected rather
than silently losing the 30 seconds. A duration must also be a non-empty
string (not a number, not `false`), strictly positive, and a valid
`time.ParseDuration` value — `""`, `"abc"`, `"-5m"`, `"0m"`, `25`, and
`false` are all rejected with a specific reason, not silently coerced or
ignored.

### `style` is reserved, not active yet

```lua
style = {
	theme = "default",
}
```

`style.theme` is accepted and validated (it must be a non-empty string,
and no other field is currently allowed in `style`), but it has **no
visual effect yet** — there's no theming system to apply it to. It exists
now so a config someone writes today doesn't need to change shape once
theming exists later.

### Schema version

Every config file must include `version = 1`. There's no default to fall
back to if it's missing — an unversioned file is rejected with a clear
message, the same as a `version` newer than this build of Gomodoro
understands (which is reported with a migration hint rather than
partially applied or silently ignored). Unknown fields anywhere in the
file are rejected too, with a "did you mean" suggestion when a close match
exists:

```text
gomodoro: config.lua: field "timer.short_braek" is unknown; did you mean "short_break"?
```

Validation is all-or-nothing: if anything in the file is invalid, nothing
from it is applied — Gomodoro will not start with half of a broken config.

### Locations

Gomodoro looks for `config.lua` using [`os.UserConfigDir()`](https://pkg.go.dev/os#UserConfigDir)
(the Go standard library's own per-platform resolution — Gomodoro never
hardcodes a path or a separator):

| System | Path |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/gomodoro/config.lua`, or `$HOME/.config/gomodoro/config.lua` |
| macOS | `$HOME/Library/Application Support/gomodoro/config.lua` |
| Windows | `%AppData%\gomodoro\config.lua` |

**A missing file at this default location is not an error** — Gomodoro
just uses the compiled-in defaults. Gomodoro never looks for a
`config.lua` in the current directory, so running it from inside some
unrelated project folder can't accidentally pick up a stray config file
there.

An explicit config file, on the other hand — via `--config` or
`GOMODORO_CONFIG` — is expected to exist: if it's missing, unreadable, or
invalid, that's an error, and Gomodoro does not start the timer, alerts,
or raw terminal mode.

### Selecting a config file explicitly

```bash
gomodoro --config /path/to/config.lua start
gomodoro start --config /path/to/config.lua
```

`--config` works on either side of `start` (it's a global option, not a
per-run duration/alert flag). `GOMODORO_CONFIG=/path/to/config.lua` does
the same thing via an environment variable. Precedence for **locating**
the file:

```text
--config  >  GOMODORO_CONFIG  >  default location
```

...and, separately, precedence for each individual **value** once a file
is found:

```text
explicit flag (-work, -sound, ...)  >  config file  >  compiled default
```

For example, with this config file:

```lua
return {version = 1, alerts = {sound = true}}
```

running `gomodoro -sound=false` plays no sound (the flag wins); running
`gomodoro` plays sound (the file's value applies, since no flag overrode
it).

### `gomodoro config` commands

```bash
gomodoro config path    # print the resolved config path and exit
gomodoro config init    # create it with defaults, if it doesn't exist yet
gomodoro config check   # validate it (including every required module)
```

None of these three start the timer, alerts, or raw terminal mode.

- **`config path`** only resolves and prints the path (respecting
  `--config`/`GOMODORO_CONFIG`) — it doesn't read or create anything.
- **`config init`** creates the config directory if needed and writes a
  single-file, commented `config.lua` containing the real defaults, using
  an atomic write (a temporary file, written and closed, then renamed into
  place — nothing partially-written is ever left as `config.lua`, and the
  temporary file is removed if any step fails). It refuses to overwrite an
  existing file.
- **`config check`** resolves the file, loads it (running every
  `require()`'d module), and validates the whole schema, printing `OK` on
  success or the specific error otherwise — with a non-zero exit code on
  failure. This is the one config command that works even when the config
  file is broken, on purpose, so you can find out *why* it's broken
  without also being blocked from finding out.

`--help`, `help`, and `--version` all work even with a broken config file
too — they're checked before any config file is touched.

## Flags

| Flag        | Description                     | Default |
| ----------- | -------------------------------- | ------- |
| `-work`     | Work session duration (minutes, must be > 0) | 25      |
| `-sbreak`   | Short break duration (minutes, must be > 0)  | 5       |
| `-lbreak`   | Long break duration (minutes, must be > 0)   | 15      |
| `-notify`   | Show a desktop notification at the end of each phase | true |
| `-sound`    | Play a short alert sound at the end of each phase | true |
| `--config`  | Path to a config file (works before or after `start`) | resolved via [Locations](#locations) |

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

Gomodoro's own code and its four dependencies
([`gen2brain/beeep`](https://github.com/gen2brain/beeep) for notifications,
[`ebitengine/oto`](https://github.com/ebitengine/oto) for sound,
[`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) for raw-mode
input, [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua) for the
config file) build without cgo on Windows, Linux, and macOS, for both
`amd64` and `arm64`. Being honest about what that does and doesn't prove:

| Platform | Compiled (cross-compilation) | Automated tests | Manually run | Notes |
| --- | --- | --- | --- | --- |
| `windows/amd64` | ✅ | ✅ | ✅ (real run: alerts fire, config load/init/check/require()/precedence all confirmed against real files) | Primary development platform |
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
platform yet. The config file's *directory resolution* on Linux/macOS
(`os.UserConfigDir()`'s XDG/`Library/Application Support` behavior) is
based on the Go standard library's own documentation, not on running
Gomodoro there — the loading/`require()`/sandbox/validation logic itself
is platform-independent pure Go and is covered by the automated test
suite on every platform it's compiled for. Linux and macOS sound/notify
runtime behavior is based on reading the dependencies' own source and
documentation, not on running Gomodoro there. If you're on Linux
(especially Hyprland/PipeWire), the [validation steps](#validating-on-linuxhyprland)
below are exactly how to close that gap.

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

# 12. Config file -- confirm the real default location on your system,
#     then create and validate it:
./gomodoro config path
# should print something under $XDG_CONFIG_HOME/gomodoro/ or
# $HOME/.config/gomodoro/
./gomodoro config init
cat "$(./gomodoro config path)"
./gomodoro config check

# 13. Edit the file it just created, e.g. set alerts.sound = false, then:
./gomodoro config check
./gomodoro start -work=1
# should be silent even though sound defaults to true, confirming the
# config file is actually being read

# 14. Flag beats config file:
./gomodoro start -work=1 -sound=true
# should be audible again even with alerts.sound = false in the file

# 15. Modular config -- split settings/timer.lua out, require() it from
#     config.lua, and confirm `config check` still resolves it correctly
#     relative to the config file's own directory, not your shell's cwd:
mkdir -p /tmp/gomodoro-modular-test/settings
cd /tmp/gomodoro-modular-test
cat > config.lua <<'LUA'
local timer = require("settings.timer")
return {version = 1, timer = timer}
LUA
cat > settings/timer.lua <<'LUA'
return {work = "1m"}
LUA
cd /
gomodoro config check --config /tmp/gomodoro-modular-test/config.lua
# must succeed even though we're not in /tmp/gomodoro-modular-test anymore

# 16. A broken config should block the timer but not `config check`:
echo 'return { version = , }' > /tmp/gomodoro-broken.lua
gomodoro start --config /tmp/gomodoro-broken.lua; echo "exit: $?"   # non-zero, no timer started
gomodoro config check --config /tmp/gomodoro-broken.lua; echo "exit: $?"  # reports the error, also non-zero
gomodoro --help; echo "exit: $?"   # still works despite the broken file above
```

Please report back:
- whether step 2 was actually audible, and through which backend
  (`pactl`/`pw-cli` output helps identify that);
- whether pause/resume in step 6 genuinely froze and resumed without
  losing or gaining time;
- whether your shell's input behaves normally after each exit path in
  step 9 (this is the real test of whether raw mode was cleanly restored);
- whether `os.UserConfigDir()` resolved to what you'd expect in step 12
  (i.e. whether `$XDG_CONFIG_HOME` is respected on your setup);
- whether step 15's `require()` really resolved relative to the config
  file's directory rather than your shell's current directory.

These are the pieces of this feature that could not be confirmed from this
environment (no real terminal, no real audio device attached here, and
`os.UserConfigDir()`'s Linux/macOS behavior only exercised via a faked
directory in automated tests, not your actual `$XDG_CONFIG_HOME`).

## Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- Pause/resume/skip only work in a real terminal (both stdin and stdout);
  redirected or piped runs fall back to the plain countdown with no
  interactive controls, by design (see [Interactive controls](#interactive-controls)).
- The config file has no hot reload — it's read once at startup, and
  `gomodoro config check` is the way to validate an edit before the next run.
- Schema errors report the file and field path, but not always a line
  number: genuine Lua syntax/runtime errors do carry a real line number
  (Lua's own parser tracks it), but Gomodoro's own schema validation (type
  and field checks on the table a config file returned) happens after Lua
  has already finished running, and gopher-lua doesn't expose per-value
  source position without the `debug` library, which the sandbox
  deliberately never opens.
- No session history or statistics — a skipped phase is deliberately *not*
  counted the same as a completed one internally, so this distinction is
  preserved for whenever history/stats are added, but nothing is persisted
  or reported yet.
- No color themes; `style.theme` is accepted and validated but has no
  visual effect yet.
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

- [ ] Continuous/looping execution (e.g. `--loop`)
- [ ] Customizable terminal color themes (via the reserved `style` section)
- [ ] `gomodoro config show` / `--effective` (print the fully-resolved config)
- [ ] Lap count / session history

## License

GPLv3 — see [LICENSE](LICENSE).
