package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
)

// layoutMobileMain shows one navigation surface at a time. Phones do not
// have enough width for the desktop rail, chat list and conversation panes.
func (u *UI) layoutMobileMain(gtx C) D {
	sz := gtx.Constraints.Max
	// Conversations have their own Back action in the header. Give the
	// message list and composer the entire viewport, especially with the
	// on-screen keyboard open.
	inChat := u.selected != nil && u.selPage == u.page && u.page != pageStatus && u.page != pageSettings
	navH := 0
	if !inChat {
		navH = min(sz.Y, gtx.Dp(64))
	}
	content := gtx
	content.Constraints = layout.Exact(image.Pt(max(0, sz.X), max(0, sz.Y-navH)))
	fillRect(content, image.Rectangle{Max: content.Constraints.Max}, u.pal.Panel)
	if inChat {
		u.layoutRightPane(content)
	} else {
		u.layoutPageSidebar(content)
	}
	if navH == 0 {
		return D{Size: sz}
	}
	nav := gtx
	nav.Constraints = layout.Exact(image.Pt(sz.X, navH))
	t := op.Offset(image.Pt(0, sz.Y-navH)).Push(gtx.Ops)
	u.layoutMobileNavigation(nav)
	t.Pop()
	return D{Size: sz}
}

func (u *UI) layoutMobileNavigation(gtx C) D {
	p := u.pal
	unread := 0
	for _, c := range u.chats {
		if c.Unread > 0 && !c.Archived {
			unread++
		}
	}
	statusUnseen := false
	for _, t := range u.statuses {
		if !t.Mine && !t.Viewed() && t.Last().Time.After(u.statusSeen) {
			statusUnseen = true
			break
		}
	}
	channelUnread := false
	for _, c := range u.channels {
		if c.Unread > 0 && c.Time.After(u.channelsSeen) {
			channelUnread = true
			break
		}
	}
	fillRect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, max(1, gtx.Dp(1))), p.Divider)
	item := func(c *widget.Clickable, active bool, glyph func(gtx C, col color.NRGBA) D, badge int, dot bool) layout.FlexChild {
		return layout.Flexed(1, func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				return u.railButton(gtx, c, active, glyph, badge, dot)
			})
		})
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		item(&u.rail.chats, u.page == pageChats && !u.sidebar.showArchived,
			func(gtx C, col color.NRGBA) D { return chatsOutline(gtx, 24, col) }, unread, false),
		item(&u.rail.calls, u.page == pageCalls,
			func(gtx C, col color.NRGBA) D { return drawIcon(gtx, icCall, 24, col) }, 0, false),
		item(&u.rail.status, u.page == pageStatus,
			func(gtx C, col color.NRGBA) D { return statusIcon(gtx, 24, col, u.page == pageStatus) }, 0, statusUnseen),
		item(&u.rail.channels, u.page == pageChannels,
			func(gtx C, col color.NRGBA) D {
				return channelsIcon(gtx, 24, col, p.RailActive, u.page == pageChannels)
			}, 0, channelUnread),
		item(&u.rail.communities, u.page == pageCommunities,
			func(gtx C, col color.NRGBA) D { return drawIcon(gtx, icGroupsLine, 24, col) }, 0, false),
		item(&u.rail.profile, u.page == pageSettings,
			func(gtx C, col color.NRGBA) D { return drawIcon(gtx, icSettings, 24, col) }, 0, false),
	)
}
