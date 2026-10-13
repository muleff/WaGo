package ui

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"image"
)

// Keep desktop geometry out of mobile content pages.
func (u *UI) contentInset(in layout.Inset) layout.Inset {
	if u.mobile {
		in.Left = min(in.Left, 16)
		in.Right = min(in.Right, 16)
	}
	return in
}

func (u *UI) mobileListGeometry(g listGeom) listGeom {
	if u.mobile {
		g.hoverLeft, g.hoverRight = 8, 8
		if g.textLeft <= 16 {
			g.iconCenter, g.textLeft = 0, 8
		} else {
			g.iconCenter, g.textLeft = 26, 52
		}
		g.padRight = 12
		g.height = max(g.height, 48)
		g.subHeight = max(g.subHeight, 48)
	}
	return g
}

// NewContext translates past system insets. Paint the reserved pixels too,
// while keeping all interactive content inside the safe area.
func (u *UI) paintSystemInsets(gtx C, e app.FrameEvent) {
	if !u.mobile {
		return
	}
	off := op.Offset(image.Pt(-e.Metric.Dp(e.Insets.Left), -e.Metric.Dp(e.Insets.Top))).Push(gtx.Ops)
	w, h := e.Size.X, e.Size.Y
	left, right := e.Metric.Dp(e.Insets.Left), e.Metric.Dp(e.Insets.Right)
	top, bottom := e.Metric.Dp(e.Insets.Top), e.Metric.Dp(e.Insets.Bottom)
	for _, r := range []image.Rectangle{image.Rect(0, 0, w, top), image.Rect(0, h-bottom, w, h), image.Rect(0, top, left, h-bottom), image.Rect(w-right, top, w, h-bottom)} {
		if !r.Empty() {
			fillRect(gtx, r, u.pal.Panel)
		}
	}
	off.Pop()
}
