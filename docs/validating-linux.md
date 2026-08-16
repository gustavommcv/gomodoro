# Validating on Linux/Hyprland

See [Platform support](platform-support.md) for why this exists: only
`windows/amd64` has been run manually so far, so this is the script that
closes the gap on a real Linux/Hyprland setup.

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
