package ui

import (
	"context"
	"testing"
	"time"
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

func TestPublishVoiceResultWakeAfterPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan recordResult, 1)
	wakeup := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		publishVoiceResult(ctx, results, func() { wakeup <- struct{}{} }, recordResult{starting: true}, func() {
			t.Error("a claimed result must not be cleaned up")
		})
	}()
	select {
	case <-wakeup:
	case <-time.After(time.Second * 2):
		t.Fatal("the UI was never notified")
	}
	select {
	case result := <-results:
		if !result.starting || result.claimed == nil {
			t.Fatalf("unexpected published result: %#v", result)
		}
		close(result.claimed)
		cancel() // Normal UI completion cancels the recording context.
	case <-time.After(time.Second):
		t.Fatal("notification ran before a result was ready")
	}
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("sender did not stop after acknowledgement")
	}
}

func TestPublishVoiceResultCancelBeforeClaim(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan recordResult, 1)
	wakeup := make(chan struct{}, 1)
	cleaned := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		publishVoiceResult(ctx, results, func() { wakeup <- struct{}{} }, recordResult{}, func() {
			close(cleaned)
		})
	}()
	select {
	case <-wakeup:
	case <-time.After(time.Second * 2):
		cancel()
		t.Fatal("the result was not published")
	}
	cancel() // Discard while the UI hasn't consumed the result.
	select {
	case <-cleaned:
	case <-time.After(time.Second * 2):
		t.Fatal("unclaimed result wasn't cleaned up on cancellation")
	}
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("sender did not exit after cancellation")
	}
}
