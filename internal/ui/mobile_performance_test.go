package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
)

// Measure CPU layout only, at a phone's logical width and 2x density.
// GPU rendering and physical-device frame rate require device measurements.
func BenchmarkMobileFrame(b *testing.B) {
	for _, view := range []string{"chats", "conversation", "updates"} {
		b.Run(view, func(b *testing.B) {
			u := New(mock.NewReference())
			u.mobile = true
			u.Start(func() {})
			u.applyEvents()
			u.SetConn(model.ConnEvent{State: model.StateOnline})
			if view == "conversation" {
				u.SelectID("test@g.us")
			} else if view == "updates" {
				u.setPage(pageStatus)
			}
			var ops op.Ops
			var router input.Router
			frame := func(now time.Time) {
				ops.Reset()
				g := layout.Context{Ops: &ops, Source: router.Source(), Now: now,
					Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Exact(image.Pt(720, 1440))}
				u.Layout(g)
				router.Frame(&ops)
			}
			now := testNow()
			for i := range 60 {
				frame(now.Add(time.Duration(i) * time.Second / 60))
			}
			now = now.Add(time.Second)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				now = now.Add(time.Second / 60)
				frame(now)
			}
		})
	}
}
