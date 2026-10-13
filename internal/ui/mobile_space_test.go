package ui

import (
	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"os"
	"testing"
	"time"
)

func TestMobileListKeepsRoomForText(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(320, 100))}
	var button widget.Clickable
	width := 0
	u.layoutListItem(gtx, &button, listItem{ic: icInfo, content: func(g C) D { width = g.Constraints.Max.X; return D{Size: image.Pt(width, 20)} }}, infoGeom)
	if width < 240 {
		t.Fatalf("text squeezed to %d px in 320 px screen", width)
	}
}

func TestMobileSwipeRepliesWithoutTreatingVerticalScrollAsReply(t *testing.T) {
	for _, tc := range []struct {
		name          string
		end           f32.Point
		cancel, reply bool
	}{
		{"right", f32.Pt(160, 20), false, true}, {"short", f32.Pt(55, 20), false, false},
		{"vertical", f32.Pt(65, 100), false, false}, {"left", f32.Pt(5, 20), false, false}, {"cancel", f32.Pt(160, 20), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := New(mock.New())
			u.mobile = true
			u.Start(func() {})
			u.applyEvents()
			u.SelectID("rina")
			c := u.selected
			m := &model.Message{ID: "swipe-test", ChatID: c.ID, Text: "Hola", Time: time.Now()}
			var ops op.Ops
			var router input.Router
			frame := func() {
				ops.Reset()
				g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(320, 180)}}
				u.layoutMessageRow(g, c, convRow{kind: rowMessage, msg: m, first: true}, 280, 8)
				router.Frame(&ops)
			}
			frame()
			router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(40, 20)})
			frame()
			router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: tc.end})
			frame()
			frame()
			kind := pointer.Release
			if tc.cancel {
				kind = pointer.Cancel
			}
			router.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, PointerID: 1, Position: tc.end})
			frame()
			frame()
			if got := u.conv.reply == m; got != tc.reply {
				t.Fatalf("reply=%v want %v", got, tc.reply)
			}
		})
	}
}

func TestMobileAllPagesFit(t *testing.T) {
	for _, size := range []image.Point{image.Pt(320, 640), image.Pt(320, 280), image.Pt(240, 320)} {
		for _, page := range []page{pageChats, pageStatus, pageCommunities, pageCalls, pageSettings} {
			u := New(mock.NewReference())
			u.mobile = true
			u.Start(func() {})
			u.applyEvents()
			u.SetConn(model.ConnEvent{State: model.StateOnline})
			u.setPage(page)
			var ops op.Ops
			var router input.Router
			frame := func() {
				ops.Reset()
				g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1.4}, Constraints: layout.Exact(size)}
				if d := u.Layout(g); d.Size != size {
					t.Fatalf("page %v size %v: %v", page, size, d.Size)
				}
				router.Frame(&ops)
			}
			frame()
			frame()
			if page == pageSettings {
				for k := range settingLogout {
					u.openSettings(k)
					frame()
					frame()
				}
			}
		}
	}
}

func TestMobileSystemInsetBackground(t *testing.T) {
	if os.Getenv("WAGO_SCREENSHOT_DIR") == "" {
		t.Skip("requires headless EGL")
	}
	u := New(mock.New())
	u.mobile = true
	u.Start(func() {})
	u.applyEvents()
	h := &host{u: u, o: Options{Mobile: true}, idle: time.NewTimer(time.Hour)}
	defer h.idle.Stop()
	win, err := headless.NewWindow(320, 640)
	if err != nil {
		t.Fatal(err)
	}
	defer win.Release()
	e := app.FrameEvent{Size: image.Pt(320, 640), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Insets: app.Insets{Top: 24, Bottom: 24}, Now: time.Now(), Frame: func(ops *op.Ops) {
		if err := win.Frame(ops); err != nil {
			t.Fatal(err)
		}
	}}
	h.windowEvent(e)
	img := image.NewRGBA(image.Rect(0, 0, 320, 640))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	for _, y := range []int{5, 630} {
		got := img.RGBAAt(10, y)
		want := u.pal.Panel
		if got.R != want.R || got.G != want.G || got.B != want.B {
			t.Fatalf("system inset at y=%d is %v, want panel %v", y, got, want)
		}
	}
}

func TestMobileReplySwipePreservesVerticalListScroll(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	u.Start(func() {})
	u.applyEvents()
	u.SelectID("rina")
	var ops op.Ops
	var router input.Router
	var list widget.List
	list.Axis = layout.Vertical
	frame := func() {
		ops.Reset()
		g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(320, 240))}
		u.scrollList(g, &list, 20, func(g C, i int) D {
			m := &model.Message{ID: itoa(i), ChatID: u.selected.ID, Text: "Hola"}
			return u.layoutMessageRow(g, u.selected, convRow{kind: rowMessage, msg: m, first: true}, 280, 8)
		})
		router.Frame(&ops)
	}
	frame()
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, 150)})
	frame()
	for _, y := range []float32{130, 100, 70, 40} {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, y)})
		frame()
	}
	router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, 40)})
	frame()
	if list.Position.First == 0 && list.Position.Offset <= 0 {
		t.Fatalf("touch scroll did not advance: %+v", list.Position)
	}
	if u.conv.reply != nil {
		t.Fatal("vertical scroll started reply")
	}
}
