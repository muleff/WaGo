package voicerec

import (
	"context"
	"testing"
)

func TestFFmpegDoesNotOpenConsoleWindow(t *testing.T) {
	cmd := ffmpegCommand(context.Background(), "ffmpeg.exe", "-version")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow ||
		cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("FFmpeg subprocess must not create a console window")
	}
}
