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

func TestWindowsAudioDevicesFFmpeg9(t *testing.T) {
	// This is FFmpeg 9's real output format. It no longer includes the
	// "DirectShow audio devices" heading which older versions printed.
	output := `[in#0 @ 0000020134a62a00] "HP TrueVision HD Camera" (video)
[in#0 @ 0000020134a62a00]   Alternative name "@camera"
[in#0 @ 0000020134a62a00] "OBS Virtual Camera" (none)
[in#0 @ 0000020134a62a00]   Alternative name "@virtual-camera"
[in#0 @ 0000020134a62a00] "Varios micrófonos (Intel® Smart Sound Technology for Digital Microphones)" (audio)
[in#0 @ 0000020134a62a00]   Alternative name "@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\wave_{14E7F6AE-9F00-4ECA-AF62-C04D398078C4}"
Error opening input file dummy.`
	devices := parseWindowsAudioDevices(output)
	if len(devices) != 1 {
		t.Fatalf("devices = %#v, want one audio device", devices)
	}
	if want := "Varios micrófonos (Intel® Smart Sound Technology for Digital Microphones)"; devices[0].name != want {
		t.Fatalf("name = %q, want %q", devices[0].name, want)
	}
	want := `audio=@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\wave_{14E7F6AE-9F00-4ECA-AF62-C04D398078C4}`
	if got := windowsCaptureArgs(devices[0]); !reflect.DeepEqual(got, []string{"-f", "dshow", "-i", want}) {
		t.Fatalf("capture args = %q, want dshow input %q", got, want)
	}
}

func TestWindowsAudioDevicesLegacyAlternates(t *testing.T) {
	devices := parseWindowsAudioDevices(`[dshow @ 0x12] DirectShow video devices
[dshow @ 0x12]  "Camera"
[dshow @ 0x12]     Alternative name "@video"
[dshow @ 0x12] DirectShow audio devices
[dshow @ 0x12]  "Microphone"
[dshow @ 0x12]     Alternative name "@audio"
[dshow @ 0x12]  "Another Microphone"`)
	if len(devices) != 2 || devices[0].alternative != "@audio" || devices[1].alternative != "" {
		t.Fatalf("devices = %#v, expected two microphones with first alternate", devices)
	}
	if got := windowsCaptureArgs(devices[1]); !reflect.DeepEqual(got, []string{"-f", "dshow", "-i", "audio=Another Microphone"}) {
		t.Fatalf("capture args = %q, expected friendly-name fallback", got)
	}
}

func TestWindowsAudioDevicesNoMicrophone(t *testing.T) {
	output := `[in#0 @ 0x12] "Camera" (video)
[in#0 @ 0x12]   Alternative name "@video"
Error opening input file dummy.`
	if got := parseWindowsAudioDevices(output); len(got) != 0 {
		t.Fatalf("unexpected microphones: %#v", got)
	}
	if !isDeviceEnumerationOutput(output) || isDeviceEnumerationOutput("Unknown input format: 'dshow'") {
		t.Fatal("failed to distinguish no microphones from unsupported DirectShow")
	}
}

func TestQuotedDeviceNameFormats(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{`"USB Microphone" (audio)`, "USB Microphone"},
		{`'Headset Mic' (audio)`, "Headset Mic"},
		{`“Micrófono integrado” (audio)`, "Micrófono integrado"},
		{`Alternative name "@device_cm_1"`, "@device_cm_1"},
	} {
		if got, ok := quotedDeviceName(tc.line); !ok || got != tc.name {
			t.Errorf("quotedDeviceName(%q) = %q, %v; want %q", tc.line, got, ok, tc.name)
		}
	}
}

func TestDirectShowSupportAndPermissionClassification(t *testing.T) {
	if !supportsDirectShow([]byte("Devices:\n D. = Demuxing supported\n D  dshow DirectShow capture\n")) ||
		supportsDirectShow([]byte("Devices:\n D  avfoundation AVFoundation\n")) {
		t.Fatal("incorrect DirectShow device support detection")
	}
	if !microphoneAccessDenied("I/O error: Access is denied (0x80070005)") ||
		microphoneAccessDenied("Error opening input file dummy") {
		t.Fatal("incorrect microphone permission classification")
	}
}
