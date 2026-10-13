//go:build !android

package voicerec

import (
	"context"
	"errors"
	"runtime"
)

// startNative is only implemented on Android (see capture_android.go).
// Other platforms record through FFmpeg (see Start).
func startNative(_ context.Context) (*Recorder, error) {
	return nil, errors.New("microphone recording is unsupported on " + runtime.GOOS)
}
