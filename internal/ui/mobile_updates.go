package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"github.com/muleff/WaGo/internal/model"
	"image"
	"strings"
)

func (u *UI) layoutMobileUpdates(gtx C) D {
	if u.status.add.Clicked(gtx) {
		u.openStatusAdd()
	}
	if u.btn("mobile:settings").Clicked(gtx) {
		u.setPage(pageSettings)
	}
	if u.btn("mobile:updates:search").Clicked(gtx) {
		u.mobileUpdatesSearch = !u.mobileUpdatesSearch
		if u.mobileUpdatesSearch {
			u.requestFocus(&u.channel.search)
		} else {
			u.channel.search.SetText("")
		}
	}
	if u.btn("mobile:explore").Clicked(gtx) {
		u.setPage(pageChannels)
	}
	q := u.channel.search.Text()
	var rows []layout.Widget
	rows = append(rows, u.layoutMobileStatusCards, func(gtx C) D {
		return layout.Inset{Left: 16, Right: 16, Top: 18, Bottom: 10}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, u.label(21, u.locale.Text("Channels"), u.pal.Text, labelOpts{weight: font.SemiBold, maxLines: 1}).Layout),
				layout.Rigid(func(gtx C) D {
					return clickable(gtx, u.btn("mobile:explore"), func(gtx C) D {

						return background(gtx, u.pal.Search, 20, func(gtx C) D {
							return layout.Inset{Left: 14, Right: 14, Top: 10, Bottom: 10}.Layout(gtx, u.label(14, u.locale.Text("Explore"), u.pal.Text, labelOpts{weight: font.Medium, maxLines: 1}).Layout)
						})
					})
				}),
			)
		})
	})
	for _, ch := range u.channels {
		if q != "" && !strings.Contains(strings.ToLower(ch.Name), strings.ToLower(q)) {
			continue
		}
		rows = append(rows, func(gtx C) D { return u.channelRow(gtx, ch) })
	}
	if len(u.channels) == 0 {
		rows = append(rows, func(gtx C) D {
			return layout.Inset{Left: 16, Right: 16, Top: 8, Bottom: 16}.Layout(gtx, u.label(16, u.locale.Text("Follow channels to see their updates here."), u.pal.TextSecondary, labelOpts{maxLines: 0}).Layout)
		})
	}
	for _, ch := range u.suggested {
		rows = append(rows, func(gtx C) D { return u.suggestedRow(gtx, ch) })
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return u.pageHeader(gtx, u.locale.Text("Updates"), u.headerButton(u.btn("mobile:updates:search"), icSearch, 24), u.headerButton(u.btn("mobile:settings"), icMenu, 24))
		}),
		layout.Rigid(func(gtx C) D {

			if !u.mobileUpdatesSearch {
				return D{}
			}
			return u.searchBox(gtx, &u.channel.search, u.locale.Text("Search"))
		}),
		layout.Flexed(1, func(gtx C) D {
			return u.scrollList(gtx, &u.status.list, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
		}),
	)
}

func (u *UI) layoutMobileStatusCards(gtx C) D {
	var mine *model.StatusThread
	threads := []*model.StatusThread{nil}
	for _, t := range u.statuses {
		if len(t.Updates) == 0 {
			continue
		}
		if t.Mine {
			mine = t
		} else {
			threads = append(threads, t)
		}
	}
	threads[0] = mine
	return layout.Inset{Left: 12, Right: 12, Top: 4, Bottom: 4}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.Y = gtx.Dp(154)
		gtx.Constraints.Max.Y = gtx.Dp(154)
		return u.status.cards.List.Layout(gtx, len(threads), func(gtx C, i int) D {
			t := threads[i]
			key := "mobile:status:add"
			if t != nil {
				key = "mobile:status:" + t.ID
			}
			c := u.btn(key)
			if c.Clicked(gtx) {
				if t != nil {
					u.status.viewer.show(t)
				} else {
					u.openStatusAdd()
				}
				gtx.Execute(op.InvalidateCmd{})
			}
			return layout.Inset{Right: 8}.Layout(gtx, func(gtx C) D {
				sz := image.Pt(gtx.Dp(96), gtx.Dp(150))
				gtx.Constraints = layout.Exact(sz)
				return clickable(gtx, c, func(gtx C) D {
					defer clip.RRect{Rect: image.Rectangle{Max: sz}, NW: gtx.Dp(16), NE: gtx.Dp(16), SW: gtx.Dp(16), SE: gtx.Dp(16)}.Push(gtx.Ops).Pop()
					bg := u.pal.Search
					label := u.locale.Text("Add status")
					if t != nil {
						label = t.Name
						if i == 0 {
							label = u.locale.Text("status.myStatus")
						}
						up := t.Last()
						if up.Background != 0 {
							bg = argbColor(up.Background)
						}
					}
					fillRect(gtx, image.Rectangle{Max: sz}, bg)
					if t != nil {
						up := t.Last()
						if len(up.Thumb) > 0 {
							thumb := up.Thumb
							e := u.images.get("st-card:"+up.ID, sz.X, func() []byte { return thumb })
							if e.state == imgReady {
								paintCover(gtx, e.op, e.size, image.Rectangle{Max: sz})
							}
						} else if up.Text != "" {
							off := op.Offset(image.Pt(gtx.Dp(6), gtx.Dp(64))).Push(gtx.Ops)
							g := gtx
							g.Constraints = layout.Constraints{Max: image.Pt(sz.X-gtx.Dp(12), gtx.Dp(34))}
							u.label(12, up.Text, rgb(0xffffff), labelOpts{maxLines: 2, align: text.Middle}).Layout(g)
							off.Pop()
						}
					}
					off := op.Offset(image.Pt(gtx.Dp(10), gtx.Dp(10))).Push(gtx.Ops)
					u.mobileStatusAvatar(gtx, t, i == 0)
					off.Pop()
					footer := image.Rect(0, sz.Y-gtx.Dp(46), sz.X, sz.Y)
					fillRect(gtx, footer, argb(0x0b141a, 0xd0))
					off = op.Offset(image.Pt(gtx.Dp(8), sz.Y-gtx.Dp(40))).Push(gtx.Ops)
					g := gtx
					g.Constraints = layout.Constraints{Max: image.Pt(sz.X-gtx.Dp(16), gtx.Dp(38))}
					u.label(13, label, rgb(0xffffff), labelOpts{weight: font.SemiBold, maxLines: 2}).Layout(g)
					off.Pop()
					return D{Size: sz}
				})
			})
		})
	})
}

func (u *UI) mobileStatusAvatar(gtx C, t *model.StatusThread, mine bool) {
	id, name, group := u.meID, u.meName(), false
	ring := u.pal.Green
	if t != nil {
		id, name, group = t.ID, t.Name, t.Group
		if t.Viewed() {
			ring = u.pal.RingViewed
		}
	}
	if mine {
		id, name, group = u.meID, u.meName(), false
	}
	size := gtx.Dp(48)
	mid := image.Pt(size/2, size/2)
	fillCircle(gtx, mid, size/2, ring)
	fillCircle(gtx, mid, size/2-gtx.Dp(2), u.pal.Panel)
	off := op.Offset(image.Pt(gtx.Dp(4), gtx.Dp(4))).Push(gtx.Ops)
	u.avatar(gtx, id, name, group, 40)
	off.Pop()
	if mine {
		r := gtx.Dp(10)
		at := image.Pt(size-gtx.Dp(3), size-gtx.Dp(3))
		fillCircle(gtx, at, r+gtx.Dp(1), u.pal.Panel)
		fillCircle(gtx, at, r, u.pal.Green)
		off := op.Offset(at.Sub(image.Pt(gtx.Dp(7), gtx.Dp(7)))).Push(gtx.Ops)
		drawIcon(gtx, icAdd, 14, u.pal.OnGreen)
		off.Pop()
	}
}
