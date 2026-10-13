package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
)

func TestMobileGroupPictureLeavesTextGap(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		u := New(mock.New())
		u.mobile = true
		var ops op.Ops
		g := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: scale, PxPerSp: scale}, Constraints: layout.Exact(image.Pt(int(320*scale), int(100*scale)))}
		var click widget.Clickable
		width := 0
		u.layoutListItem(g, &click, listItem{
			glyph:   func(g C, _ color.NRGBA) D { return D{Size: image.Pt(g.Dp(52), g.Dp(52))} },
			content: func(g C) D { width = g.Constraints.Max.X; return D{Size: image.Pt(width, g.Dp(20))} },
		}, infoGeom)
		// 8dp margins, 52dp picture, 12dp gap, 12dp trailing padding.
		if width > int(228*scale) || width < int(220*scale) {
			t.Fatalf("density %g: text width %d, picture needs a 12dp gap", scale, width)
		}
	}
}

func TestMobileChatsMenuCanOpenNewGroupByTouch(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		u := New(mock.NewReference())
		u.mobile = true
		u.Start(func() {})
		u.applyEvents()
		u.SetConn(model.ConnEvent{State: model.StateOnline})
		var ops op.Ops
		var router input.Router
		now := testNow()
		frame := func() {
			ops.Reset()
			g := layout.Context{Ops: &ops, Source: router.Source(), Now: now, Metric: unit.Metric{PxPerDp: scale, PxPerSp: scale}, Constraints: layout.Exact(image.Pt(int(320*scale), int(640*scale)))}
			u.Layout(g)
			router.Frame(&ops)
			now = now.Add(time.Second)
		}
		frame()
		menuButton := f32.Pt(296*scale, 28*scale)
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: menuButton})
		frame()
		router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: menuButton})
		frame()
		frame()
		if !u.menu.open {
			t.Fatal("Chats three-dot button did not open the menu")
		}
		if u.menu.anchor.X < int(200*scale) {
			t.Fatalf("mobile menu has no visible anchor: %v", u.menu.anchor)
		}
		at := f32.Pt(float32(u.menu.anchor.X)-50*scale, float32(u.menu.anchor.Y)+28*scale)
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: at})
		frame()
		router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: at})
		frame()
		frame()
		if u.newChat.step != ncMembers || u.menu.open {
			t.Fatalf("density %g: Chats menu cannot open New group by touch (anchor %v)", scale, u.menu.anchor)
		}
	}
}

func TestSpanishAttachmentMenuLabels(t *testing.T) {
	u := New(mock.New())
	u.setLanguage("es")
	for _, it := range u.attachMenuItems(&model.Chat{ID: "chat"}) {
		if it.label != u.locale.Text(it.label) {
			t.Errorf("attachment menu still displays English: %q", it.label)
		}
	}
}

func TestMobileMenuStartsCommunityCreation(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	u.Start(func() {})
	u.applyEvents()
	u.menu.open = true
	u.btn("mobile:menu:community").Click()
	var ops op.Ops
	u.updateMenu(layout.Context{Ops: &ops, Now: testNow()})
	if u.newChat.step != ncGroup || u.newChat.base != ncGroup || u.menu.open {
		t.Fatal("New community must open its name/photo form and close the menu")
	}
	u.newChat.name.SetText("Familia")
	u.createGroup()
	u.applyEvents()
	if u.newChat.open() || u.page != pageCommunities {
		t.Fatalf("community creation did not return to Communities: page %v", u.page)
	}
	found := false
	for _, cm := range u.communities {
		found = found || cm.Name == "Familia" && cm.Announcements != ""
	}
	if !found {
		t.Fatal("created community has no visible announcement group")
	}
	u.openNewGroup(nil)
	if u.newChat.community {
		t.Fatal("community mode leaked into New group")
	}
}

func TestMobileAttachmentSheetContactAction(t *testing.T) {
	u := New(mock.NewReference())
	u.mobile = true
	u.Start(func() {})
	u.applyEvents()
	u.SelectID("test@g.us")
	u.mouse = image.Pt(290, 590)
	u.openAttachMenu()
	var ops op.Ops
	var router input.Router
	now := testNow()
	frame := func() {
		ops.Reset()
		g := layout.Context{Ops: &ops, Source: router.Source(), Now: now, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(360, 640))}
		u.layoutCtxMenu(g)
		router.Frame(&ops)
		now = now.Add(time.Second)
	}
	frame()
	frame()
	// Contact is the fourth cell: second row on a 320dp device, but first
	// row's last cell on this 360dp device. The sheet ends above the composer.
	point := f32.Pt(304, 400)
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: point})
	frame()
	router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: point})
	frame()
	frame()
	if !u.dialog.contacts {
		t.Fatalf("visible Contact attachment did not open contact picker: %v", u.dialog.kind)
	}
}

func TestMobileStickerNameHeaderIsCompact(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	var ops op.Ops
	g := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(688, 1000)}}
	head := u.stickerHeader(g, &model.Message{Sender: "Yosue", SenderID: "yosue"}, g.Dp(150), g.Dp(320))
	if head.size.X >= g.Dp(100) {
		t.Fatalf("short sender name still has a sticker-wide header: %v", head.size)
	}
}
