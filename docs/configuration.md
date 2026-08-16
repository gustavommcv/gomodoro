# Configuration

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

## A single file is enough

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

## Or split it into modules

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

## `require()` resolution

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

## Sandbox

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

## Defaults

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

## Durations

Durations are plain `time.ParseDuration`-style strings (`"25m"`, `"1h"`,
`"90s"`), with one additional rule: **they must amount to a whole number
of minutes.** Gomodoro's timer itself only operates in whole minutes today
(that didn't change with this feature), so `"25m30s"` is rejected rather
than silently losing the 30 seconds. A duration must also be a non-empty
string (not a number, not `false`), strictly positive, and a valid
`time.ParseDuration` value — `""`, `"abc"`, `"-5m"`, `"0m"`, `25`, and
`false` are all rejected with a specific reason, not silently coerced or
ignored.

## `style` is reserved, not active yet

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

## Schema version

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

## Locations

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

## Selecting a config file explicitly

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

## `gomodoro config` commands

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
