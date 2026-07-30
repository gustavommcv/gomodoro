package sound

import (
	"math"
	"testing"
)

func decodeSamples(t *testing.T, buf []byte) []float32 {
	t.Helper()
	if len(buf)%bytesPerSample != 0 {
		t.Fatalf("buffer length %d is not a multiple of %d bytes per sample", len(buf), bytesPerSample)
	}
	samples := make([]float32, len(buf)/bytesPerSample)
	for i := range samples {
		bits := uint32(buf[i*4]) | uint32(buf[i*4+1])<<8 | uint32(buf[i*4+2])<<16 | uint32(buf[i*4+3])<<24
		samples[i] = math.Float32frombits(bits)
	}
	return samples
}

func TestGenerateTone(t *testing.T) {
	buf := generateTone()

	t.Run("is not empty", func(t *testing.T) {
		if len(buf) == 0 {
			t.Fatal("generateTone returned an empty buffer")
		}
	})

	t.Run("length is a whole number of 32-bit float samples", func(t *testing.T) {
		if len(buf)%bytesPerSample != 0 {
			t.Errorf("buffer length %d is not a multiple of %d", len(buf), bytesPerSample)
		}
	})

	t.Run("duration is approximately the configured tone duration", func(t *testing.T) {
		numSamples := len(buf) / bytesPerSample
		gotDuration := float64(numSamples) / SampleRate
		wantDuration := toneDuration.Seconds()

		// Allow a tiny tolerance for integer sample-count rounding.
		if diff := math.Abs(gotDuration - wantDuration); diff > 0.002 {
			t.Errorf("got duration %.4fs, want approximately %.4fs (diff %.4fs)", gotDuration, wantDuration, diff)
		}
	})

	t.Run("every sample decodes to a finite value within the configured amplitude", func(t *testing.T) {
		samples := decodeSamples(t, buf)
		for i, s := range samples {
			f := float64(s)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				t.Fatalf("sample %d is not finite: %v", i, f)
			}
			// A small tolerance above the nominal amplitude for float rounding.
			if math.Abs(f) > amplitude+0.001 {
				t.Errorf("sample %d amplitude %v exceeds configured amplitude %v", i, f, amplitude)
			}
		}
	})

	t.Run("fades in at the very start: the first sample is silent", func(t *testing.T) {
		samples := decodeSamples(t, buf)
		if got := math.Abs(float64(samples[0])); got > 0.0001 {
			t.Errorf("first sample amplitude = %v, want ~0 (fade-in)", got)
		}
	})

	t.Run("fades out at the very end: the last sample is silent", func(t *testing.T) {
		samples := decodeSamples(t, buf)
		last := samples[len(samples)-1]
		if got := math.Abs(float64(last)); got > 0.0001 {
			t.Errorf("last sample amplitude = %v, want ~0 (fade-out)", got)
		}
	})

	t.Run("reaches full amplitude in the sustain region, not just near zero throughout", func(t *testing.T) {
		samples := decodeSamples(t, buf)
		mid := len(samples) / 2
		// Somewhere in the middle third of the tone, the envelope should be
		// at (or very near) full amplitude at at least one sample -- this
		// guards against an envelope bug that silences the whole tone.
		peak := 0.0
		for _, s := range samples[mid-len(samples)/6 : mid+len(samples)/6] {
			if a := math.Abs(float64(s)); a > peak {
				peak = a
			}
		}
		if peak < amplitude*0.9 {
			t.Errorf("peak amplitude in sustain region = %v, want close to %v", peak, amplitude)
		}
	})

	t.Run("is deterministic across calls", func(t *testing.T) {
		other := generateTone()
		if len(other) != len(buf) {
			t.Fatalf("length differs across calls: %d vs %d", len(other), len(buf))
		}
		for i := range buf {
			if buf[i] != other[i] {
				t.Fatalf("byte %d differs across calls: %d vs %d", i, buf[i], other[i])
			}
		}
	})
}
