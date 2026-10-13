package voicerec

import (
	"github.com/tphakala/go-opus/oggopus"
)

// nativeVoiceConfig matches the FFmpeg voice-note settings (see
// opusEncodeArgs): 48 kHz mono Ogg/Opus at 96 kbit/s VBR, for recorders
// that encode in-process instead of shelling out to FFmpeg.
func nativeVoiceConfig() oggopus.Config {
	return oggopus.Config{
		SampleRate: 48000,
		Channels:   1,
		Bitrate:    96000,
		Complexity: 10,
	}
}
