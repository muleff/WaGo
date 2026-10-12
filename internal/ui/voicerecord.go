package ui

import (
	"context"
	"fmt"
	"image"
	"os"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/skidy89/openWA/internal/model"
	"github.com/skidy89/openWA/internal/voicerec"
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
	ctx, cancel := context.WithCancel(context.Background())
	r := &u.voiceRecord
	r.phase, r.chatID, r.ctx, r.cancel = recordStarting, u.selected.ID, ctx, cancel
	r.result = make(chan recordResult)
	results, notify := r.result, u.images.invalidate
	u.stopOutgoingTyping()
	u.closePicker()
	go func() {
		rec, err := voicerec.Start(ctx)
		if notify != nil {
			notify()
		}
		select {
		case results <- recordResult{starting: true, recorder: rec, err: err}:
			// The UI now owns the recorder and will stop it on navigation.
		case <-ctx.Done():
			if rec != nil {
				rec.Abort()
			}
		}
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
		if notify != nil {
			notify()
		}
		select {
		case results <- recordResult{path: path, err: err}:
		case <-ctx.Done():
			_ = os.Remove(path)
		}
	}()
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
		if result.starting {
			if result.err != nil {
				u.discardVoiceRecord()
				u.toast(result.err.Error())
				return
			}
			r.rec, r.phase, r.started = result.recorder, recordActive, u.now()
			return
		}
		chatID := r.chatID
		u.discardVoiceRecord()
		if result.err != nil {
			u.toast(result.err.Error())
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
				u.toast(err.Error())
			} else {
				u.toast("Microphone capture stopped unexpectedly.")
			}
		}
	default:
	}
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
