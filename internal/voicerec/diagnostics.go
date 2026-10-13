package voicerec

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Diagnostic details are kept in memory until the next recording attempt.
// They contain device names and FFmpeg messages, never recorded audio.
var captureDiagnostics struct {
	sync.Mutex
	entries []string
}

func resetCaptureDiagnostics() {
	captureDiagnostics.Lock()
	defer captureDiagnostics.Unlock()
	captureDiagnostics.entries = []string{
		"WaGo microphone diagnostics",
		"Time: " + time.Now().Format(time.RFC3339),
		"Operating system: " + runtime.GOOS + "/" + runtime.GOARCH,
		"Recording content is not included.",
	}
}

func captureDebug(format string, args ...any) {
	entry := fmt.Sprintf(format, args...)
	if len(entry) > 8000 {
		entry = entry[:8000] + " [truncated]"
	}
	captureDiagnostics.Lock()
	defer captureDiagnostics.Unlock()
	if len(captureDiagnostics.entries) < 70 {
		captureDiagnostics.entries = append(captureDiagnostics.entries, entry)
	}
}

// DiagnosticReport returns the latest capture attempt for explicit copying
// by the user. No microphone audio is included.
func DiagnosticReport() string {
	captureDiagnostics.Lock()
	defer captureDiagnostics.Unlock()
	return strings.Join(captureDiagnostics.entries, "\n\n")
}
