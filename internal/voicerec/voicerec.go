// Package voicerec records microphone audio as WhatsApp-compatible Ogg/Opus.
// FFmpeg is run only after the user explicitly starts a recording.
package voicerec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoMicrophone     = errors.New("no microphone was found")
	ErrMicrophoneAccess = errors.New("microphone access was denied")
)

type Recorder struct {
	path   string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
	log    bytes.Buffer
}

// Start finds an input device and begins recording. The caller must call
// Finish or Abort; the supplied context also terminates the recording.
func Start(ctx context.Context) (*Recorder, error) {
	bin, err := findFFmpeg(ctx)
	if err != nil {
		return nil, err
	}
	input, err := inputArgs(ctx, bin)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "wago-voice-*.ogg")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	_ = f.Close()
	// FFmpeg creates the final file itself; a failed launch leaves no empty file.
	_ = os.Remove(path)
	ctx, cancel := context.WithCancel(ctx)
	args := append([]string{"-hide_banner", "-loglevel", "error", "-nostats", "-y"}, input...)
	args = append(args, opusEncodeArgs(path)...)
	cmd := exec.CommandContext(ctx, bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	r := &Recorder{path: path, cmd: cmd, stdin: stdin, cancel: cancel, done: make(chan struct{})}
	cmd.Stderr = &r.log
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		cancel()
		return nil, fmt.Errorf("could not start microphone capture: %w", err)
	}
	go func() {
		err := cmd.Wait()
		r.mu.Lock()
		r.err = err
		r.mu.Unlock()
		close(r.done)
	}()
	return r, nil
}

// opusEncodeArgs uses a single high-quality encode for WhatsApp voice notes.
// Opus is inherently lossy, but 96 kbit/s variable bitrate is substantially
// more faithful to microphone input than a low-bitrate voice preset. Keep
// mono for the PTT format; never apply noise gates or normalization.
func opusEncodeArgs(path string) []string {
	return []string{
		"-vn", "-ac", "1", "-ar", "48000",
		"-c:a", "libopus", "-b:a", "96k", "-vbr", "on",
		"-compression_level", "10", "-application", "audio",
		"-f", "ogg", path,
	}
}

// findFFmpeg skips old builds without Ogg/Opus support. Some Windows installs
// leave an obsolete ffmpeg.exe earlier on PATH than the WinGet installation.
func findFFmpeg(ctx context.Context) (string, error) {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		// The Windows installer and portable ZIP ship a compatible FFmpeg next
		// to OpenWA.exe. Prefer it over an obsolete FFmpeg elsewhere on PATH.
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
		}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir != "" {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			installed, _ := filepath.Glob(filepath.Join(local, "Microsoft", "WinGet", "Packages",
				"Gyan.FFmpeg_*", "ffmpeg-*", "bin", "ffmpeg.exe"))
			candidates = append(candidates, installed...)
		}
	}
	for _, bin := range candidates {
		if _, err := os.Stat(bin); err != nil {
			continue
		}
		check, cancel := context.WithTimeout(ctx, 4*time.Second)
		out, err := exec.CommandContext(check, bin, "-hide_banner", "-encoders").CombinedOutput()
		cancel()
		if err == nil && bytes.Contains(out, []byte("libopus")) {
			if runtime.GOOS != "windows" {
				return bin, nil
			}
			// A usable encoder does not imply that this build can capture
			// DirectShow devices. Check the input driver before selecting it.
			check, stop := context.WithTimeout(ctx, 4*time.Second)
			devices, deviceErr := exec.CommandContext(check, bin, "-hide_banner", "-devices").CombinedOutput()
			stop()
			if deviceErr == nil && supportsDirectShow(devices) {
				return bin, nil
			}
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
	}
	return "", errors.New("recording needs a recent FFmpeg build with libopus; install FFmpeg and restart WaGo")
}

func (r *Recorder) Done() <-chan struct{} { return r.done }

// Error is safe once Done has closed.
func (r *Recorder) Error() error {
	<-r.done
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		return nil
	}
	msg := strings.TrimSpace(r.log.String())
	if len(msg) > 500 {
		msg = msg[len(msg)-500:]
	}
	if msg != "" {
		if microphoneAccessDenied(msg) {
			return fmt.Errorf("%w: %s", ErrMicrophoneAccess, msg)
		}
		return fmt.Errorf("microphone capture failed: %s", msg)
	}
	return fmt.Errorf("microphone capture failed: %w", r.err)
}

// Finish finalizes the Ogg stream. A truncated capture is never sent.
func (r *Recorder) Finish() (string, error) {
	select {
	case <-r.done:
	default:
		_, _ = io.WriteString(r.stdin, "q\n")
	}
	_ = r.stdin.Close()
	select {
	case <-r.done:
	case <-time.After(7 * time.Second):
		r.cancel()
		<-r.done
	}
	defer r.cancel()
	if err := r.Error(); err != nil {
		_ = os.Remove(r.path)
		return "", err
	}
	f, err := os.Open(r.path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var header [36]byte
	n, err := io.ReadFull(f, header[:])
	if err != nil || n < len(header) || string(header[:4]) != "OggS" ||
		!bytes.Contains(header[:], []byte("OpusHead")) {
		_ = os.Remove(r.path)
		return "", errors.New("the microphone did not produce a valid Ogg/Opus recording")
	}
	if st, err := f.Stat(); err != nil || st.Size() < 128 {
		_ = os.Remove(r.path)
		return "", errors.New("the recording is empty")
	}
	return r.path, nil
}

// Abort discards the recording. It can be called while FFmpeg is running.
func (r *Recorder) Abort() {
	r.cancel()
	_ = r.stdin.Close()
	<-r.done
	_ = os.Remove(r.path)
}

func inputArgs(ctx context.Context, bin string) ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		// Enumeration intentionally exits nonzero. Recent FFmpeg versions
		// identify each device with '(audio)' rather than a DirectShow heading.
		check, cancel := context.WithTimeout(ctx, 8*time.Second)
		cmd := exec.CommandContext(check, bin, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
		out, cmdErr := cmd.CombinedOutput()
		checkErr := check.Err()
		cancel()
		devices := parseWindowsAudioDevices(string(out))
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if checkErr != nil {
			return nil, fmt.Errorf("microphone discovery timed out: %w", checkErr)
		}
		if len(devices) == 0 {
			if microphoneAccessDenied(string(out)) {
				return nil, ErrMicrophoneAccess
			}
			if !isDeviceEnumerationOutput(string(out)) {
				if cmdErr != nil {
					return nil, fmt.Errorf("could not enumerate microphones: %w: %s", cmdErr, tailLog(string(out)))
				}
				return nil, fmt.Errorf("could not enumerate microphones: %s", tailLog(string(out)))
			}
			return nil, ErrNoMicrophone
		}
		return windowsCaptureArgs(devices[0]), nil
	case "darwin":
		return []string{"-f", "avfoundation", "-i", ":0"}, nil
	case "linux":
		return []string{"-f", "pulse", "-i", "default"}, nil
	default:
		return nil, fmt.Errorf("microphone recording is unsupported on %s", runtime.GOOS)
	}
}

type windowsAudioDevice struct {
	name        string
	alternative string
}

func windowsCaptureArgs(device windowsAudioDevice) []string {
	name := device.name
	if device.alternative != "" {
		name = device.alternative
	}
	return []string{"-f", "dshow", "-i", "audio=" + name}
}

// parseWindowsAudioDevices accepts both FFmpeg's older DirectShow section
// format and the format introduced in recent builds, which identifies each
// device inline as '(audio)', '(video)', or '(none)'.
func parseWindowsAudioDevices(output string) []windowsAudioDevice {
	var devices []windowsAudioDevice
	legacyAudio := false
	lastAudio := -1
	for _, line := range strings.Split(output, "\n") {
		if end := strings.IndexByte(line, ']'); end >= 0 {
			line = line[end+1:]
		}
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "DirectShow audio devices"):
			legacyAudio, lastAudio = true, -1
			continue
		case strings.HasPrefix(line, "DirectShow video devices"):
			legacyAudio, lastAudio = false, -1
			continue
		case strings.HasPrefix(line, "Alternative name"):
			if lastAudio >= 0 {
				if name, ok := quotedDeviceName(line); ok {
					devices[lastAudio].alternative = name
				}
			}
			continue
		}
		name, ok := quotedDeviceName(line)
		if !ok {
			lastAudio = -1
			continue
		}
		// FFmpeg 9+ prints '"Microphone" (audio)', without headings.
		// An explicit type wins over a previously seen legacy section.
		end := strings.LastIndexByte(line, '"')
		suffix := strings.TrimSpace(line[end+1:])
		if suffix != "" {
			lastAudio = -1
			if suffix != "(audio)" {
				continue
			}
		} else if !legacyAudio {
			lastAudio = -1
			continue
		}
		devices = append(devices, windowsAudioDevice{name: name})
		lastAudio = len(devices) - 1
	}
	return devices
}

func quotedDeviceName(line string) (string, bool) {
	start := strings.IndexByte(line, '"')
	end := strings.LastIndexByte(line, '"')
	if start < 0 || end <= start {
		return "", false
	}
	return line[start+1 : end], true
}

// windowsAudioDevices retains the friendly names for status/tests. Capture
// uses the alternative ID when present, so duplicate or localized names work.
func windowsAudioDevices(output string) []string {
	devices := parseWindowsAudioDevices(output)
	names := make([]string, 0, len(devices))
	for _, device := range devices {
		names = append(names, device.name)
	}
	return names
}

func supportsDirectShow(output []byte) bool {
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.Contains(fields[0], "D") && fields[1] == "dshow" {
			return true
		}
	}
	return false
}

func isDeviceEnumerationOutput(output string) bool {
	return strings.Contains(output, "DirectShow audio devices") ||
		strings.Contains(output, "DirectShow video devices") ||
		strings.Contains(output, " (audio)") ||
		strings.Contains(output, " (video)") ||
		strings.Contains(output, " (none)") ||
		strings.Contains(output, "Error opening input file dummy.") ||
		strings.Contains(strings.ToLower(output), "no devices found")
}

func microphoneAccessDenied(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "permission denied") ||
		strings.Contains(output, "access is denied") ||
		strings.Contains(output, "access denied") ||
		strings.Contains(output, "0x80070005")
}

func tailLog(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 350 {
		output = output[len(output)-350:]
	}
	return output
}
