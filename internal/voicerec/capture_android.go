//go:build android

package voicerec

import (
	"context"
	"fmt"
	"os"

	"github.com/gen2brain/malgo"
	"github.com/tphakala/go-opus/oggopus"
)

// startNative records the microphone without FFmpeg: malgo captures PCM
// through the OS audio backend (AAudio/OpenSL ES on Android) and the pure
// Go oggopus encoder writes WhatsApp-compatible Ogg/Opus. There is no
// external binary to ship, find or probe on mobile.
func startNative(ctx context.Context) (*Recorder, error) {
	captureDebug("Starting native microphone capture")
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		captureDebug("Audio context init failed: %v", err)
		return nil, fmt.Errorf("could not start microphone capture: %w", err)
	}
	devices, err := mctx.Devices(malgo.Capture)
	if err != nil {
		captureDebug("Audio device enumeration failed: %v", err)
		mctx.Uninit()
		mctx.Free()
		return nil, fmt.Errorf("could not enumerate microphones: %w", err)
	}
	names := make([]string, 0, len(devices))
	for i := range devices {
		names = append(names, devices[i].Name())
	}
	captureDebug("Capture devices: %q", names)
	if len(devices) == 0 {
		mctx.Uninit()
		mctx.Free()
		return nil, ErrNoMicrophone
	}
	f, err := os.CreateTemp("", "wago-voice-*.ogg")
	if err != nil {
		mctx.Uninit()
		mctx.Free()
		return nil, err
	}
	path := f.Name()
	enc, err := oggopus.NewEncoder(f, nativeVoiceConfig())
	if err != nil {
		captureDebug("Opus encoder init failed: %v", err)
		f.Close()
		os.Remove(path)
		mctx.Uninit()
		mctx.Free()
		return nil, fmt.Errorf("could not start microphone capture: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	r := &Recorder{path: path, cancel: cancel, done: make(chan struct{})}
	frames := make(chan []byte, 64)
	config := malgo.DefaultDeviceConfig(malgo.Capture)
	config.SampleRate = 48000
	config.Capture.Format = malgo.FormatS16
	config.Capture.Channels = 1
	device, err := malgo.InitDevice(mctx.Context, config, malgo.DeviceCallbacks{
		Data: func(_, input []byte, _ uint32) {
			if len(input) == 0 {
				return
			}
			buf := make([]byte, len(input))
			copy(buf, input)
			select {
			case frames <- buf:
			default:
				// The encoder fell behind; drop instead of
				// stalling the realtime audio callback.
				captureDebug("Dropped a late audio frame")
			}
		},
	})
	if err != nil {
		captureDebug("Capture device init failed: %v", err)
		enc.Close()
		f.Close()
		os.Remove(path)
		mctx.Uninit()
		mctx.Free()
		cancel()
		return nil, fmt.Errorf("could not start microphone capture: %w", err)
	}
	if err := device.Start(); err != nil {
		captureDebug("Capture device start failed: %v", err)
		device.Uninit()
		enc.Close()
		f.Close()
		os.Remove(path)
		mctx.Uninit()
		mctx.Free()
		cancel()
		return nil, fmt.Errorf("could not start microphone capture: %w", err)
	}
	go func() {
		defer close(r.done)
		defer func() {
			device.Stop()
			enc.Close()
			device.Uninit()
			mctx.Uninit()
			mctx.Free()
			f.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				if err := finishPCM(func() { device.Stop() }, frames, enc); err != nil {
					r.mu.Lock()
					r.err = err
					r.mu.Unlock()
				}
				return
			case buf := <-frames:
				if _, err := enc.Write(buf); err != nil {
					r.mu.Lock()
					r.err = err
					r.mu.Unlock()
					return
				}
			}
		}
	}()
	return r, nil
}
