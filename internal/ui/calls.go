package ui

import (
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
)

type callsState struct {
	add widget.Clickable
}

// layoutCallsList draws the Calls page. Calling isn't supported, so it
// only explains that.
func (u *UI) layoutCallsList(gtx C) D {
	p := u.pal
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return u.pageHeader(gtx, u.locale.Text("Calls"), u.headerButton(&u.calls.add, icAddCircle, 27))
		}),
		layout.Rigid(func(gtx C) D {
			return u.sectionLabel(gtx, u.locale.Text("Recent"), layout.Inset{Left: 27, Top: 20, Bottom: 22}, labelOpts{maxLines: 1})
		}),
		layout.Rigid(func(gtx C) D {
			return u.contentInset(layout.Inset{Left: 27, Right: 27}).Layout(gtx, func(gtx C) D {
				l := u.label(15.2, u.locale.Text("wip"),
					p.TextSecondary, labelOpts{align: text.Start})
				l.MaxLines = 0
				return l.Layout(gtx)
			})
		}),
	)
}
