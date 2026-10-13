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

func TestMobileLayoutsFitSmallScreenAndKeyboard(t *testing.T) {
	for _, size := range []image.Point{{X: 320, Y: 640}, {X: 320, Y: 280}, {X: 240, Y: 320}} {
		u := New(mock.New())
		u.mobile = true
		u.Start(func() {})
		u.applyEvents()
		var ops op.Ops
		var router input.Router
		frame := func() {
			t.Helper()
			ops.Reset()
			gtx := layout.Context{Ops: &ops, Now: time.Now(), Source: router.Source(),
				Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size)}
			if got := u.Layout(gtx).Size; got != size {
				t.Fatalf("mobile frame at %v: size %v", size, got)
			}
			router.Frame(&ops)
		}
		u.conn = model.ConnEvent{State: model.StateStarting}
		frame() // unpaired app, soft keyboard could be showing
		u.conn = model.ConnEvent{State: model.StateOnline}
		u.SelectID("rina")
		frame() // conversation expands to the viewport
	}
}

func TestPhonePairRequestDoesNotBlockFrame(t *testing.T) {
	u := New(mock.New())
	u.Start(func() {})
	u.requestPhonePairCode("15551234567")
	if u.login.pairing == nil {
		t.Fatal("pair code request was not dispatched")
	}
	select {
	case result := <-u.login.pairing:
		if result.err != nil || result.code != "12345678" {
			t.Fatalf("unexpected pairing response: %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pairing code result was not published")
	}
}
