package ui

import (
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/muleff/WaGo/internal/mock"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidScreenshots(t *testing.T) {
	dir := os.Getenv("WAGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set WAGO_SCREENSHOT_DIR for rendered Android UI evidence")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name string
		size image.Point
		sp   float32
	}{
		{"", image.Pt(390, 780), 1}, {"-320", image.Pt(320, 640), 1}, {"-landscape", image.Pt(780, 390), 1}, {"-keyboard", image.Pt(320, 280), 1}, {"-large-text", image.Pt(390, 780), 1.4},
	} {
		for _, view := range []string{"chats", "contact", "updates", "login"} {
			u := New(mock.NewReference())
			u.mobile = true
			u.Start(func() {})
			u.applyEvents()
			u.SetConn(model.ConnEvent{State: model.StateOnline})
			if view == "contact" {
				u.SelectID("test@g.us")
				u.ShowContact("vivy@lid", 0, 0)
				u.info.data.Name = "+50498273976"
				u.info.name = "+50498273976"
				u.info.data.Phone = "+50498273976"
				u.info.data.Business = nil
				u.info.data.About = "Disponible"
			}
			if view == "login" {
				u.SetConn(model.ConnEvent{State: model.StateStarting})
				u.verifyStep = true
				u.verifyCode.SetText("AB12-CD34")
			}
			if view == "updates" {
				u.setPage(pageStatus)
			}
			win, err := headless.NewWindow(scenario.size.X, scenario.size.Y)
			if err != nil {
				t.Fatal(err)
			}
			var ops op.Ops
			var router input.Router
			for range 3 {
				ops.Reset()
				gtx := layout.Context{Ops: &ops, Now: testNow(), Source: router.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: scenario.sp}, Constraints: layout.Exact(scenario.size)}
				u.Layout(gtx)
				router.Frame(&ops)
				if err := win.Frame(&ops); err != nil {
					win.Release()
					t.Fatal(err)
				}
			}
			img := image.NewRGBA(image.Rectangle{Max: scenario.size})
			if err := win.Screenshot(img); err != nil {
				win.Release()
				t.Fatal(err)
			}
			win.Release()
			f, err := os.Create(filepath.Join(dir, "android-"+view+scenario.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, img)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
