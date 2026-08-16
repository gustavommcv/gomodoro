# Usage

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

## Interactive controls

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

## Alerts

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
[config file](configuration.md) — an explicit flag always overrides the
config file, which always overrides the compiled-in default.

### How the sound alert works

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
platforms — see [Platform support](platform-support.md) for exactly which
combinations were compiled, tested, and run manually.

### How the visual notification works

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
