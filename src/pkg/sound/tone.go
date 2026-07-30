package sound

import (
	"math"
	"time"
)

// SampleRate is the sample rate the tone is generated at, and the rate the
// audio backend is configured to play at (44100 and 48000 are the only
// values oto guarantees not to distort; 44100 is the more universally
// supported of the two).
const SampleRate = 44100

const (
	// bytesPerSample matches oto.FormatFloat32LE: one mono channel, 32-bit
	// float, little-endian.
	bytesPerSample = 4

	// toneFreq, toneDuration and amplitude were chosen to be short,
	// discreet and non-jarring: a single gentle pitch around 350ms long,
	// well under the requested 200-700ms window, at a quarter of full
	// scale.
	toneFreq     = 660.0 // Hz (E5) -- a mid pitch, not shrill
	toneDuration = 350 * time.Millisecond
	amplitude    = 0.25

	// fadeIn and fadeOut are short linear ramps at the very start and end
	// of the tone so the first and last samples are silent -- this is what
	// avoids the click/pop a hard-edged tone would otherwise have.
	fadeIn  = 15 * time.Millisecond
	fadeOut = 60 * time.Millisecond
)

// generateTone synthesizes the short alert tone deterministically, as raw
// little-endian 32-bit float PCM samples (matching oto.FormatFloat32LE):
// a single mono sine wave with a linear fade-in and fade-out. It has no
// external inputs and always produces the same bytes, so callers generate
// it once and reuse the result rather than calling it per phase.
func generateTone() []byte {
	numSamples := int(SampleRate * toneDuration.Seconds())
	fadeInSamples := int(SampleRate * fadeIn.Seconds())
	fadeOutSamples := int(SampleRate * fadeOut.Seconds())

	buf := make([]byte, numSamples*bytesPerSample)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / SampleRate
		sample := amplitude * math.Sin(2*math.Pi*toneFreq*t)

		envelope := 1.0
		if i < fadeInSamples {
			envelope = float64(i) / float64(fadeInSamples)
		}
		if distFromEnd := numSamples - 1 - i; distFromEnd < fadeOutSamples {
			if fadeOutEnvelope := float64(distFromEnd) / float64(fadeOutSamples); fadeOutEnvelope < envelope {
				envelope = fadeOutEnvelope
			}
		}
		sample *= envelope

		bits := math.Float32bits(float32(sample))
		buf[i*bytesPerSample+0] = byte(bits)
		buf[i*bytesPerSample+1] = byte(bits >> 8)
		buf[i*bytesPerSample+2] = byte(bits >> 16)
		buf[i*bytesPerSample+3] = byte(bits >> 24)
	}
	return buf
}
