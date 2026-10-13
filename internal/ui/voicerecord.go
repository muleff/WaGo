package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"runtime"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/muleff/WaGo/internal/model"
	"github.com/muleff/WaGo/internal/voicerec"
)

type recordPhase uint8

const (
	recordIdle recordPhase = iota
	recordStarting
	recordActive
	recordFinishing
)

type recordResult struct {
	starting bool
	recorder *voicerec.Recorder
	path     string
	err      error
	claimed  chan struct{}
}

type voiceRecordState struct {
	phase   recordPhase
	chatID  string
	started time.Time
	ctx     context.Context
	cancel  context.CancelFunc
	rec     *voicerec.Recorder
	result  chan recordResult
	discard widget.Clickable
}

func (u *UI) beginVoiceRecord() {
	if u.selected == nil || u.voiceRecord.phase != recordIdle || u.ghostMode() {
		return
	}
	if u.window == nil {
		u.toast("Microphone recording requires an active app window.")
		return
	}
	if runtime.GOOS == "windows" && !u.microphoneConsent {
		u.confirm("Microphone permission",
			"WaGo wants to use your microphone to record voice messages. Audio is recorded only while you choose to record. Windows may also require microphone access to be enabled in Privacy settings.",
			dialogButton{label: "Allow microphone", primary: true, run: func() {
				u.microphoneConsent = true
				u.startVoiceRecord()
			}},
		)
		return
	}
	u.startVoiceRecord()
}

func (u *UI) startVoiceRecord() {
	if u.selected == nil || u.voiceRecord.phase != recordIdle || u.ghostMode() || u.window == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &u.voiceRecord
	r.phase, r.chatID, r.ctx, r.cancel = recordStarting, u.selected.ID, ctx, cancel
	r.result = make(chan recordResult, 1)
	results, notify := r.result, u.images.invalidate
	u.stopOutgoingTyping()
	u.closePicker()
	go func() {
		rec, err := voicerec.Start(ctx)
		publishVoiceResult(ctx, results, notify,
			recordResult{starting: true, recorder: rec, err: err}, func() {
				if rec != nil {
					rec.Abort()
				}
			})
	}()
}

func (u *UI) finishVoiceRecord() {
	r := &u.voiceRecord
	if r.phase != recordActive || r.rec == nil {
		return
	}
	if u.now().Sub(r.started) < time.Second {
		u.toast("Record at least one second of audio.")
		return
	}
	r.phase = recordFinishing
	results, rec, notify, ctx := r.result, r.rec, u.images.invalidate, r.ctx
	go func() {
		path, err := rec.Finish()
		publishVoiceResult(ctx, results, notify, recordResult{path: path, err: err}, func() {
			if path != "" {
				_ = os.Remove(path)
			}
		})
	}()
}

// Publish before requesting a frame so the UI cannot consume the wakeup
// before a sender on an unbuffered channel is ready. The acknowledgement
// transfers ownership to the UI; until then cancellation cleans the result.
func publishVoiceResult(ctx context.Context, results chan<- recordResult, notify func(), result recordResult, cleanup func()) {
	claimed := make(chan struct{})
	result.claimed = claimed
	select {
	case <-ctx.Done():
		cleanup()
		return
	case results <- result:
	}
	if notify != nil {
		notify()
	}
	select {
	case <-claimed:
	case <-ctx.Done():
		// The UI may have claimed the result immediately before canceling.
		select {
		case <-claimed:
		default:
			cleanup()
		}
	}
}

// discardVoiceRecord is also used when a chat, account or window closes.
func (u *UI) discardVoiceRecord() {
	r := &u.voiceRecord
	if r.phase == recordIdle {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	if r.phase == recordActive && r.rec != nil {
		go r.rec.Abort()
	}
	u.voiceRecord = voiceRecordState{}
}

func (u *UI) updateVoiceRecord() {
	r := &u.voiceRecord
	if r.phase == recordIdle {
		return
	}
	select {
	case result := <-r.result:
		if result.claimed != nil {
			close(result.claimed)
		}
		if result.starting {
			if result.err != nil {
				u.discardVoiceRecord()
				u.reportVoiceCaptureError(result.err)
				return
			}
			r.rec, r.phase, r.started = result.recorder, recordActive, u.now()
			return
		}
		chatID := r.chatID
		u.discardVoiceRecord()
		if result.err != nil {
			u.reportVoiceCaptureError(result.err)
			return
		}
		if result.path == "" {
			return
		}
		if u.selected == nil || u.selected.ID != chatID || u.ghostMode() {
			_ = os.Remove(result.path)
			return
		}
		u.queueSend(chatID, &attachFile{
			Attachment: model.Attachment{Path: result.path, Media: model.MediaVoice},
			temp:       true,
		}, model.Draft{Reply: u.conv.reply})
		u.conv.reply = nil
		u.flushOutbox()
		u.scrollMessages(layout.Position{})
	case <-r.recDone():
		if r.phase == recordActive {
			err := r.rec.Error()
			u.discardVoiceRecord()
			if err != nil {
				u.reportVoiceCaptureError(err)
			} else {
				u.toast("Microphone capture stopped unexpectedly.")
			}
		}
	default:
	}
}

// A classic Win32 app can ask for consent in its own UI, but cannot force
// Windows' per-app system permission prompt. When Windows denies capture,
// offer the OS microphone privacy settings instead.
func (u *UI) reportVoiceCaptureError(err error) {
	if runtime.GOOS == "windows" {
		body := "Microphone capture failed: " + err.Error()
		switch {
		case errors.Is(err, voicerec.ErrMicrophoneAccess):
			body = "Windows blocked microphone access. Enable microphone access for desktop apps in Windows Settings."
		case errors.Is(err, voicerec.ErrNoMicrophone):
			body = "FFmpeg did not detect an audio input. This may be a device or driver issue, not necessarily a missing microphone. Copy the debug details to investigate."
		}
		u.confirm("Microphone unavailable", body,
			dialogButton{
				label: "Copy debug details", primary: true,
				run: func() {
					u.pendingCopy = voicerec.DiagnosticReport()
					u.toast("Microphone diagnostics copied. Review them before sharing.")
				},
			},
			dialogButton{
				label: "Microphone settings",
				run: func() {
					if !openTarget("ms-settings:privacy-microphone") {
						u.toast("Couldn't open Windows microphone settings.")
					}
				},
			},
		)
		return
	}
	u.toast(err.Error())
}

func (r *voiceRecordState) recDone() <-chan struct{} {
	if r.phase == recordActive && r.rec != nil {
		return r.rec.Done()
	}
	return nil
}

func (u *UI) layoutVoiceComposer(gtx C) D {
	r := &u.voiceRecord
	p := u.pal
	return layout.Inset{Left: 12, Right: 12, Top: 6, Bottom: 12}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		m := op.Record(gtx.Ops)
		dims := layout.Inset{Left: 8, Right: 8}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return u.iconButton(gtx, &r.discard, icDelete, 40, 23, p.IconStrong)
				}),
				layout.Rigid(layout.Spacer{Width: 8}.Layout),
				layout.Flexed(1, func(gtx C) D {
					status := "Preparing microphone…"
					if r.phase == recordFinishing {
						status = "Sending recording…"
					} else if r.phase == recordActive {
						elapsed := max(0, int(u.now().Sub(r.started).Seconds()))
						status = fmt.Sprintf("● Recording  %d:%02d", elapsed/60, elapsed%60)
						gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second)})
					}
					return vcenter(gtx, gtx.Dp(55), u.label(15, status, p.Text, labelOpts{weight: font.Medium}).Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return clickable(gtx, &u.conv.send, func(gtx C) D {
						sz := gtx.Dp(40)
						mid := image.Pt(sz/2, sz/2)
						fillCircle(gtx, mid, sz/2, p.Green)
						centerIn(gtx, sz, iconW(icSend, 21, p.OnGreen))
						return D{Size: image.Pt(sz, sz)}
					})
				}),
			)
		})
		call := m.Stop()
		fillRRect(gtx, image.Rectangle{Max: dims.Size}, gtx.Dp(28), p.Composer)
		call.Add(gtx.Ops)
		u.conv.composerH = dims.Size.Y + gtx.Dp(18)
		return dims
	})
}
