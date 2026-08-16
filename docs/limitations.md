# Current limitations

- Runs exactly one lap and exits — no continuous/looping mode yet.
- Pause/resume/skip only work in a real terminal (both stdin and stdout);
  redirected or piped runs fall back to the plain countdown with no
  interactive controls, by design (see [Interactive controls](usage.md#interactive-controls)).
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
  [Platform support](platform-support.md)) — only through the automated
  test suite, which drives the same logic without a real terminal.
