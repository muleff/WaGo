package voicerec

import (
	"context"
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// FFmpeg is a console program. Starting it from a GUI executable without
// CREATE_NO_WINDOW causes a visible terminal to flash on every probe.
func ffmpegCommand(ctx context.Context, bin string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
}
