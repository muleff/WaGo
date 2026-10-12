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

var ErrNoMicrophone = errors.New("no microphone was found")

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
			return bin, nil
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
		// DShow uses the device's display name, not a universal "default" alias.
		cmd := exec.CommandContext(ctx, bin, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
		out, _ := cmd.CombinedOutput() // enumeration exits nonzero by design
		devices := windowsAudioDevices(string(out))
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, ErrNoMicrophone
		}
		return []string{"-f", "dshow", "-i", "audio=" + devices[0]}, nil
	case "darwin":
		return []string{"-f", "avfoundation", "-i", ":0"}, nil
	case "linux":
		return []string{"-f", "pulse", "-i", "default"}, nil
	default:
		return nil, fmt.Errorf("microphone recording is unsupported on %s", runtime.GOOS)
	}
}

// windowsAudioDevices reads the quoted names in FFmpeg's DirectShow audio
// section, skipping the alternate device identifiers and all video devices.
func windowsAudioDevices(output string) []string {
	var names []string
	audio := false
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.Contains(line, "DirectShow audio devices"):
			audio = true
			continue
		case strings.Contains(line, "DirectShow video devices"):
			audio = false
			continue
		}
		if !audio || strings.Contains(line, "Alternative name") {
			continue
		}
		start := strings.IndexByte(line, '"')
		end := strings.LastIndexByte(line, '"')
		if start >= 0 && end > start {
			names = append(names, line[start+1:end])
		}
	}
	return names
}
