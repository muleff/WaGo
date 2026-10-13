package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/text"
)

// Attachment actions share one bounded sheet and the desktop action keys.
func (u *UI) mobileAttachPanel(gtx C, items []menuItem) D {
	w := max(0, gtx.Constraints.Max.X-gtx.Dp(16))
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	cols := 4
	if w < gtx.Dp(340) {
		cols = 3
	}
	return background(gtx, u.pal.Menu, 20, func(gtx C) D {
		return layout.Inset{Top: 12, Bottom: 12, Left: 8, Right: 8}.Layout(gtx, func(gtx C) D {
			children := []layout.FlexChild{layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					fillRRect(gtx, image.Rect(0, 0, gtx.Dp(32), gtx.Dp(4)), gtx.Dp(2), u.pal.TextSecondary)
					return D{Size: image.Pt(gtx.Dp(32), gtx.Dp(4))}
				})
			}), layout.Rigid(layout.Spacer{Height: 12}.Layout)}
			for first := 0; first < len(items); first += cols {
				first := first
				children = append(children, layout.Rigid(func(gtx C) D {
					var cells []layout.FlexChild
					for j := range cols {
						index := first + j
						cells = append(cells, layout.Flexed(1, func(gtx C) D {
							if index >= len(items) {
								return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(88))}
							}
							it := items[index]
							gtx.Constraints.Min.Y = gtx.Dp(88)
							return clickable(gtx, u.btn("menu:"+it.key), func(gtx C) D {
								return layout.Center.Layout(gtx, func(gtx C) D {
									return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx C) D {
											w, h := gtx.Dp(56), gtx.Dp(40)
											fillRRect(gtx, image.Rect(0, 0, w, h), gtx.Dp(20), u.pal.Panel)
											gtx.Constraints = layout.Exact(image.Pt(w, h))
											return layout.Center.Layout(gtx, iconW(it.ic, 27, it.col))
										}), layout.Rigid(layout.Spacer{Height: 6}.Layout),
										layout.Rigid(u.label(12.5, it.label, u.pal.TextSecondary, labelOpts{maxLines: 2, align: text.Middle}).Layout))
								})
							})
						}))
					}
					return layout.Flex{}.Layout(gtx, cells...)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}
