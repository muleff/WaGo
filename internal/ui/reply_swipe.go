package ui

import (
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"math"
	"time"
)

type replySwipe struct {
	settle                        float32
	released                      time.Time
	start                         f32.Point
	id                            pointer.ID
	pressed, horizontal, rejected bool
	offset                        float32
}

func (u *UI) layoutReplySwipe(gtx C, c *model.Chat, m *model.Message, lay layout.Widget) D {
	if !u.mobile || m == nil || u.conv.selecting || m.Kind == model.KindDeleted || !m.Revoked.IsZero() || isChannelID(c.ID) || u.sendBlocked(c) != "" {
		return lay(gtx)
	}
	key := "swipe:" + c.ID + ":" + m.ID
	u.btn(key)
	s := &u.clicks.m[key].swipe
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: s, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e := ev.(pointer.Event)
		switch e.Kind {
		case pointer.Press:
			if e.Source != pointer.Touch || s.pressed {
				continue
			}
			*s = replySwipe{start: e.Position, id: e.PointerID, pressed: true}
		case pointer.Drag:
			if !s.pressed || e.PointerID != s.id || s.rejected {
				continue
			}
			dx, dy := e.Position.X-s.start.X, e.Position.Y-s.start.Y
			if !s.horizontal {
				if math.Abs(float64(dy)) > float64(gtx.Dp(10)) && math.Abs(float64(dy)) >= math.Abs(float64(dx)) {
					s.rejected = true
					continue
				}
				if dx > float32(gtx.Dp(12)) && dx > 1.5*float32(math.Abs(float64(dy))) {
					s.horizontal = true
					gtx.Execute(pointer.GrabCmd{Tag: s, ID: e.PointerID})
				}
			}
			if s.horizontal {
				s.offset = min(float32(gtx.Dp(80)), max(0, dx))
				gtx.Execute(op.InvalidateCmd{})
			}
		case pointer.Release:
			if s.pressed && e.PointerID == s.id {
				if s.horizontal && s.offset >= float32(gtx.Dp(56)) {
					u.startReply(m)
				}
				*s = replySwipe{settle: s.offset, released: gtx.Now}
				gtx.Execute(op.InvalidateCmd{})
			}
		case pointer.Cancel:
			*s = replySwipe{}
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	offset := s.offset
	if !s.pressed && s.settle > 0 {
		t := min(float32(1), max(float32(0), float32(gtx.Now.Sub(s.released))/float32(160*time.Millisecond)))
		offset = s.settle * (1 - easeOut(t))
		if t >= 1 {
			s.settle = 0
		} else {
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	shift := int(offset)
	if shift > 0 {
		t := op.Offset(image.Pt(gtx.Dp(12), gtx.Dp(8))).Push(gtx.Ops)
		col := u.pal.TextSecondary
		if shift >= gtx.Dp(56) {
			col = u.pal.Green
		}
		drawIcon(gtx, icReply, 22, col)
		t.Pop()
	}
	t := op.Offset(image.Pt(shift, 0)).Push(gtx.Ops)
	d := lay(gtx)
	t.Pop()
	// The hit area stays stationary while the message follows the finger.
	area := clip.Rect{Max: d.Size}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, s)
	pass.Pop()
	area.Pop()
	return d
}
