package ui

import (
	"context"
	"testing"
)

func TestVoiceRecordingStopsOnNavigation(t *testing.T) {
	st := newSlashTest(t, "rina")
	ctx, cancel := context.WithCancel(context.Background())
	st.u.voiceRecord = voiceRecordState{phase: recordStarting, ctx: ctx, cancel: cancel}
	st.u.setPage(pageSettings)
	if st.u.voiceRecord.phase != recordIdle {
		t.Fatal("recording is still active after navigating away from chats")
	}
	if ctx.Err() == nil {
		t.Fatal("microphone context was not canceled")
	}
}

func TestVoiceRecordingStopsOnAccountShutdown(t *testing.T) {
	st := newSlashTest(t, "rina")
	ctx, cancel := context.WithCancel(context.Background())
	st.u.voiceRecord = voiceRecordState{phase: recordStarting, ctx: ctx, cancel: cancel}
	st.u.shutdown()
	if st.u.voiceRecord.phase != recordIdle {
		t.Fatal("recording is still active after account shutdown")
	}
	if ctx.Err() == nil {
		t.Fatal("microphone context was not canceled on shutdown")
	}
}
