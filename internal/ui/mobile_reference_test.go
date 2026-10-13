package ui

import (
	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"testing"
	"time"
)

func TestMobileChatRowClickableAtScreenEdge(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	var ops op.Ops
	var router input.Router
	var click widget.Clickable
	clicked := false
	c := &model.Chat{ID: "edge", Name: "Contacto"}
	frame := func() {
		ops.Reset()
		g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(360, 84))}
		clicked = clicked || click.Clicked(g)
		u.chatRow(g, c, rowOpts{click: &click})
		router.Frame(&ops)
	}
	frame()
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(4, 30)})
	frame()
	router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(4, 30)})
	frame()
	if !clicked {
		t.Fatal("desktop side gutter blocks tapping a mobile chat row")
	}
}

func TestSpanishPresenceKeepsClockAndNames(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	u.setLanguage("es")
	for _, tc := range []struct{ in, want string }{
		{"online", "en línea"}, {"last seen today at 08:30", "últ. vez hoy a las 8:30 a. m."},
		{"last seen yesterday at 20:30", "últ. vez ayer a las 8:30 p. m."},
		{"last seen 31/10/2026 at 00:05", "últ. vez 31/10/2026 a las 12:05 a. m."},
	} {
		if got := u.presenceText(tc.in); got != tc.want {
			t.Errorf("%q: %q want %q", tc.in, got, tc.want)
		}
	}
	c := &model.Chat{IsGroup: true, Typing: []model.Typist{{Name: "online"}}}
	if got := u.typingText(c); got != "online está escribiendo…" {
		t.Fatalf("contact name translated: %q", got)
	}
}

func TestMobileStatusCardsScrollHorizontally(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	u.Start(func() {})
	u.applyEvents()
	for i := range 8 {
		u.statuses = append(u.statuses, &model.StatusThread{ID: itoa(i), Name: "Estado", Updates: []*model.StatusUpdate{{ID: itoa(i), Text: "Hola", Background: 0xff005c4b}}})
	}
	var ops op.Ops
	var router input.Router
	frame := func() {
		ops.Reset()
		g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(360, 180)}}
		u.layoutMobileStatusCards(g)
		router.Frame(&ops)
	}
	frame()
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(260, 90)})
	frame()
	for _, x := range []float32{220, 180, 140, 100} {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(x, 90)})
		frame()
	}
	router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, 90)})
	frame()
	if p := u.status.cards.Position; p.First == 0 && p.Offset <= 0 {
		t.Fatalf("status carousel did not move: %+v", p)
	}
	if u.status.viewer.isOpen() {
		t.Fatal("swipe opened a status")
	}
}

func TestSpanishMediaLabelsPreserveUserCaptions(t *testing.T) {
	u := New(mock.New())
	u.setLanguage("es")
	if got := u.mediaText(&model.Message{Media: model.MediaImage, Text: "Photo"}); got != "Photo" {
		t.Fatalf("user caption translated: %q", got)
	}
	if got := u.mediaText(&model.Message{Media: model.MediaImage}); got != "Foto" {
		t.Fatalf("app media label untranslated: %q", got)
	}
}

func TestMobileUpdatesSearchAndExploreButtons(t *testing.T) {
	for _, tc := range []struct {
		name string
		pos  f32.Point
	}{{"search", f32.Pt(272, 32)}, {"explore", f32.Pt(310, 267)}} {
		t.Run(tc.name, func(t *testing.T) {
			u := New(mock.NewReference())
			u.mobile = true
			u.Start(func() {})
			u.applyEvents()
			u.SetConn(model.ConnEvent{State: model.StateOnline})
			u.setPage(pageStatus)
			var ops op.Ops
			var router input.Router
			frame := func() {
				ops.Reset()
				g := layout.Context{Ops: &ops, Source: router.Source(), Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(360, 720))}
				u.Layout(g)
				router.Frame(&ops)
			}
			frame()
			frame()
			router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: tc.pos})
			frame()
			router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: tc.pos})
			frame()
			frame()
			if tc.name == "search" && !u.mobileUpdatesSearch {
				t.Fatal("search button did not open the field")
			}
			if tc.name == "explore" && u.page != pageChannels {
				t.Fatalf("explore page=%v", u.page)
			}
		})
	}
}
