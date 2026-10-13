package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
	"github.com/muleff/WaGo/internal/model"
)

func (u *UI) layoutMobileUpdates(gtx C) D {
	p := u.pal
	if u.status.add.Clicked(gtx) {
		u.openStatusAdd()
	}
	if u.btn("mobile:settings").Clicked(gtx) {
		u.setPage(pageSettings)
	}
	var mine *model.StatusThread
	var rows []layout.Widget
	heading := func(label string) layout.Widget {
		return func(gtx C) D {
			return u.sectionLabel(gtx, u.locale.Text(label), layout.Inset{Left: 20, Top: 20, Bottom: 10}, labelOpts{weight: font.SemiBold, maxLines: 1})
		}
	}
	for _, t := range u.statuses {
		if t.Mine {
			mine = t
			break
		}
	}
	rows = append(rows, heading("Status"), func(gtx C) D {
		sub := u.locale.Text("status.click.update")
		if mine != nil {
			sub = statusTime(mine.Last().Time, u.now(), u)
		}
		return u.statusRow(gtx, "status:me", mine, u.locale.Text("status.myStatus"), sub, 76, true)
	})
	for _, viewed := range []bool{false, true} {
		added := false
		for _, t := range u.statuses {
			if t.Mine || t.Viewed() != viewed {
				continue
			}
			if !added {
				label := "Recent"
				if viewed {
					label = "Viewed"
				}
				rows = append(rows, heading(label))
				added = true
			}
			rows = append(rows, func(gtx C) D {
				return u.statusRow(gtx, "status:"+t.ID, t, t.Name, statusTime(t.Last().Time, u.now(), u), 76, false)
			})
		}
	}
	rows = append(rows, heading("Channels"))
	for _, ch := range u.channels {
		rows = append(rows, func(gtx C) D { return u.channelRow(gtx, ch) })
	}
	if len(u.channels) == 0 {
		rows = append(rows, func(gtx C) D {
			return layout.Inset{Left: 20, Right: 20, Top: 8, Bottom: 16}.Layout(gtx, u.label(16, u.locale.Text("Follow channels to see their updates here."), p.TextSecondary, labelOpts{maxLines: 0}).Layout)
		})
	}
	for _, ch := range u.suggested {
		rows = append(rows, func(gtx C) D { return u.suggestedRow(gtx, ch) })
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return u.pageHeader(gtx, u.locale.Text("Updates"), u.headerButton(&u.status.add, icAddCircle, 26), u.headerButton(u.btn("mobile:settings"), icSettings, 24))
		}),
		layout.Flexed(1, func(gtx C) D {
			return u.scrollList(gtx, &u.status.list, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
		}))
}
