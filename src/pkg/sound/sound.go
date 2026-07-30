// Package sound plays a short, procedurally generated alert tone through
// the operating system's normal audio output (WASAPI on Windows, an
// AudioQueue on macOS, PulseAudio/PipeWire-Pulse or ALSA on Linux/BSD --
// all via github.com/ebitengine/oto, without cgo). It deliberately exposes
// a single function, Play, so the rest of Gomodoro can call it without
// knowing anything about PCM formats, audio contexts, or the underlying
// library.
package sound

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	// readyTimeout bounds how long Play waits for the audio backend to
	// report it is ready. oto's readiness channel has no built-in timeout
	// of its own and can block forever if no usable audio device ever
	// appears, so this is what keeps a missing audio server from hanging
	// the whole program.
	readyTimeout = 3 * time.Second

	// playTimeout bounds how long Play waits for the tone to finish, on
	// top of the tone's own nominal duration. It exists for the same
	// reason as readyTimeout: to fail with a clear error instead of
	// blocking forever if playback never completes.
	playTimeout = 1 * time.Second

	// pollInterval is how often Play checks whether the tone has finished
	// playing. It is not the mechanism that ends playback -- oto's own
	// Player.IsPlaying is -- it is just how often that check happens.
	pollInterval = 5 * time.Millisecond
)

// backend lazily creates and remembers the single process-wide oto.Context.
// oto forbids creating more than one context per process -- even a second
// attempt after a failed first one returns an error -- so this must run at
// most once no matter how many times Play is called, and a failed attempt
// cannot be retried within the same process run.
var (
	backendOnce sync.Once
	backendCtx  *oto.Context
	backendErr  error
)

func backend() (*oto.Context, error) {
	backendOnce.Do(func() {
		actx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   SampleRate,
			ChannelCount: 1,
			Format:       oto.FormatFloat32LE,
		})
		if err != nil {
			backendErr = fmt.Errorf("initialize audio backend: %w", err)
			return
		}

		select {
		case <-ready:
			backendCtx = actx
		case <-time.After(readyTimeout):
			backendErr = errors.New("initialize audio backend: timed out waiting for an audio device")
		}
	})
	return backendCtx, backendErr
}

// toneOnce/toneBuf cache the generated PCM tone so it is computed once and
// reused for every Play call, not regenerated per phase.
var (
	toneOnce sync.Once
	toneBuf  []byte
)

func tone() []byte {
	toneOnce.Do(func() { toneBuf = generateTone() })
	return toneBuf
}

// Play plays the alert tone through the system's normal audio output and
// returns once playback has finished (confirmed via the player's own
// IsPlaying state, not an arbitrary sleep). It returns a descriptive,
// honest error -- never a false success -- if the audio backend could not
// be initialized or playback did not complete in time; it never falls back
// to the terminal bell or any other substitute. If ctx is cancelled before
// or during playback, Play stops waiting and returns ctx.Err().
func Play(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	actx, err := backend()
	if err != nil {
		return err
	}

	p := actx.NewPlayer(bytes.NewReader(tone()))
	p.Play()

	deadline := time.NewTimer(toneDuration + playTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for p.IsPlaying() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("play sound: timed out waiting for playback to finish")
		case <-ticker.C:
		}
	}

	// The player has no reference kept elsewhere once Play returns; without
	// this, nothing stops the garbage collector from reclaiming it (and,
	// per oto's own docs, whatever that would do to in-flight playback)
	// before this point. By the time we get here playback has already been
	// confirmed finished, so this only guards against a GC running in the
	// narrow window right before that confirmation.
	runtime.KeepAlive(p)

	return nil
}
