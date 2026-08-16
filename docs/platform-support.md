# Platform support

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
(especially Hyprland/PipeWire), the [validation steps](validating-linux.md)
are exactly how to close that gap.
