package ui

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/muleff/WaGo/internal/filepick"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"testing"
	"time"
)

func TestPhoneDigitsUseTextFont(t *testing.T) {
	u := New(mock.New())
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(1000, 100)}}
	d := u.label(20, "+50498273976", u.pal.Text).Layout(gtx)
	if d.Size.X > 170 {
		t.Fatalf("phone digits too wide: %dpx at 20sp", d.Size.X)
	}
}
func TestMobileEscapeClosesInfoBeforeChat(t *testing.T) {
	u := New(mock.NewReference())
	u.mobile = true
	u.Start(func() {})
	u.SelectID("test@g.us")
	u.openInfo("test@g.us")
	u.escape()
	if u.info.open {
		t.Fatal("Back left contact panel open")
	}
	if u.selected == nil {
		t.Fatal("Back closed conversation before contact panel")
	}
}
func TestPickedTemporaryFilesResetPicking(t *testing.T) {
	u := New(mock.New())
	u.attach.picking = true
	u.attach.results = make(chan pickResult, 1)
	u.attach.results <- pickResult{temp: true, picked: true, chatID: "missing"}
	u.updateAttach()
	if u.attach.picking {
		t.Fatal("temporary picker result left selection locked")
	}
}

func TestMobileNavigationHasFourLabeledDestinations(t *testing.T) {
	tabs := mobileTabs()
	if len(tabs) != 4 {
		t.Fatalf("tabs=%d, want 4", len(tabs))
	}
	for i, want := range []string{"Chats", "Updates", "Communities", "Calls"} {
		if tabs[i].label != want {
			t.Errorf("tab %d=%q", i, tabs[i].label)
		}
	}
}

func TestMobileBackAtRootIsDelegated(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	if u.mobileBack() {
		t.Fatal("root Back consumed")
	}
	u.page = pageStatus
	if !u.mobileBack() || u.page != pageChats {
		t.Fatal("Back did not return to chats")
	}
}

func TestContactQuickActionsLocalized(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	if got := u.locale.Text("Share"); got != "Compartir" {
		t.Fatalf("share=%q", got)
	}
	if got := u.label(16, "Contact info", u.pal.Text).Text; got != "Contact info" {
		t.Fatalf("user message translated to %q", got)
	}
}

func TestMobileCalendarCopyIsSpanish(t *testing.T) {
	u := New(mock.New())
	if got := u.calendarText("13 October 2026"); got != "13 octubre 2026" {
		t.Fatalf("date=%q", got)
	}
}
func TestPasteDoesNotUnlockActivePicker(t *testing.T) {
	u := New(mock.New())
	u.attach.picking = true
	u.attach.results = make(chan pickResult, 1)
	u.attach.results <- pickResult{temp: true, chatID: "missing"}
	u.updateAttach()
	if !u.attach.picking {
		t.Fatal("paste unlocked an active picker")
	}
}

func TestMobileOfflineSendKeepsDraft(t *testing.T) {
	u := New(mock.New())
	u.mobile = true
	u.Start(func() {})
	u.SelectID("rina")
	u.conn = model.ConnEvent{State: model.StateStarting}
	u.conv.composer.SetText("mensaje pendiente")
	u.sendComposer()
	if got := u.conv.composer.Text(); got != "mensaje pendiente" {
		t.Fatalf("offline draft lost: %q", got)
	}
}

func TestSAFMetadataControlsAttachmentType(t *testing.T) {
	f := classifySelection(filepick.Selection{Path: "/cache/pick.bin", Name: "photo.backup", MIME: "image/jpeg; charset=utf-8"}, false)
	if f.Media != model.MediaImage || f.MIME != "image/jpeg" || f.Name != "photo.backup" {
		t.Fatalf("metadata lost: %+v", f)
	}
	if d := classifySelection(filepick.Selection{Path: "/cache/pick.bin", MIME: "image/jpeg"}, true); d.Media != model.MediaDocument {
		t.Fatal("document picker changed to photo")
	}
}
