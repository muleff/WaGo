package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
)

func (u *UI) layoutMobileFilters(gtx C) D {
	u.sidebar.hiddenFilters = nil
	u.sidebar.chipSel.step(gtx, u.sidebar.filter, durSwitch)
	return layout.Inset{Left: 12, Right: 12, Top: 4, Bottom: 14}.Layout(gtx, func(gtx C) D {
		return u.sidebar.chipList.List.Layout(gtx, len(filterNames), func(gtx C, i int) D {
			active := u.sidebar.chipSel.of(i)
			col := mix(u.pal.ChipText, u.pal.ChipActiveText, active)
			return layout.Inset{Right: 8}.Layout(gtx, func(gtx C) D {
				return u.chip(gtx, &u.sidebar.chips[i], active, func(gtx C) D {
					return layout.Inset{Left: 14, Right: 14}.Layout(gtx, u.label(14, u.locale.Text(filterNames[i]), col, labelOpts{weight: font.Medium, maxLines: 1}).Layout)
				})
			})
		})
	})
}
