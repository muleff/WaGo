package video

import (
	"math"
	"os"
	"testing"

	"github.com/tphakala/go-opus/oggopus"
)

// TestScanVoiceLength encodes a second of synthetic PCM and checks the
// scanner recovers its exact length, which the player shows as duration.
func TestScanVoiceLength(t *testing.T) {
	f, err := os.CreateTemp("", "wago-scan-test-*.ogg")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	enc, err := oggopus.NewEncoder(f, oggopus.Config{
		SampleRate: 48000,
		Channels:   1,
		Bitrate:    96000,
		Complexity: 10,
	})
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	const samples = 48000
	pcm := make([]byte, 2*samples)
	for i := range samples {
		v := int16(16000 * math.Sin(2*math.Pi*440*float64(i)/48000))
		pcm[2*i] = byte(v)
		pcm[2*i+1] = byte(v >> 8)
	}
	if _, err := enc.Write(pcm); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	f, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	total, ch, err := scanVoice(f)
	if err != nil {
		t.Fatal(err)
	}
	if ch != 1 || total != samples {
		t.Fatalf("scan = %d samples x %d ch, want %d x 1", total, ch, samples)
	}
}

func TestScanVoiceRejectsNonOpus(t *testing.T) {
	f, err := os.CreateTemp("", "wago-scan-test-*.ogg")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString("not ogg opus"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	f, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, _, err := scanVoice(f); err == nil {
		t.Fatal("non-Opus passed voice scan")
	}
}
