package voicerec

import (
	"reflect"
	"slices"
	"testing"
)

func TestHighQualityVoiceEncoding(t *testing.T) {
	args := opusEncodeArgs("test.ogg")
	for _, pair := range [][2]string{
		{"-c:a", "libopus"},
		{"-b:a", "96k"},
		{"-ar", "48000"},
		{"-vbr", "on"},
		{"-application", "audio"},
		{"-f", "ogg"},
	} {
		i := slices.Index(args, pair[0])
		if i < 0 || i+1 >= len(args) || args[i+1] != pair[1] {
			t.Errorf("missing %s %s in %q", pair[0], pair[1], args)
		}
	}
	if !slices.Contains(args, "test.ogg") {
		t.Fatalf("missing output file in %q", args)
	}
}

func TestWindowsAudioDevices(t *testing.T) {
	output := `[dshow @ 0x123] DirectShow video devices (some may be both video and audio devices)
[dshow @ 0x123]  "Camera"
[dshow @ 0x123]     Alternative name "@device_video"
[dshow @ 0x123] DirectShow audio devices
[dshow @ 0x123]  "Microphone (Realtek Audio)"
[dshow @ 0x123]     Alternative name "@device_cm_1"
[dshow @ 0x123]  "USB Microphone"
[dshow @ 0x123]     Alternative name "@device_cm_2"`
	want := []string{"Microphone (Realtek Audio)", "USB Microphone"}
	if got := windowsAudioDevices(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("devices = %q, want %q", got, want)
	}
}

func TestWindowsAudioDevicesEmpty(t *testing.T) {
	if got := windowsAudioDevices("[dshow] DirectShow video devices\n[dshow] \"Camera\""); len(got) != 0 {
		t.Fatalf("unexpected audio devices: %q", got)
	}
}
