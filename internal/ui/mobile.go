package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"image"
)

type mobileTab struct {
	page  page
	label string
}

func mobileTabs() []mobileTab {
	return []mobileTab{{pageChats, "Chats"}, {pageStatus, "Updates"}, {pageCommunities, "Communities"}, {pageCalls, "Calls"}}
}

func (u *UI) layoutMobileMain(gtx C) D {
	sz := gtx.Constraints.Max
	inChat := u.selected != nil && u.selPage == u.page && u.page != pageStatus && u.page != pageSettings
	sendingStatus := u.page == pageStatus && isStatusDestination(u.attach.chatID) && len(u.attach.files) > 0
	navH := 0
	if !inChat && !sendingStatus && !(u.page == pageSettings && u.settings.detail != 0) && !u.newChat.open() {
		navH = min(sz.Y, gtx.Dp(72))
	}
	content := gtx
	content.Constraints = layout.Exact(image.Pt(sz.X, max(0, sz.Y-navH)))
	fillRect(content, image.Rectangle{Max: content.Constraints.Max}, u.pal.Panel)
	switch {
	case inChat || sendingStatus:
		u.layoutRightPane(content)
	case u.page == pageStatus:
		u.layoutMobileUpdates(content)
	default:
		u.layoutPageSidebar(content)
	}
	if navH > 0 && !inChat && (u.page == pageChats || u.page == pageStatus) && !u.newChat.open() {
		size := gtx.Dp(56)
		t := op.Offset(image.Pt(max(0, sz.X-size-gtx.Dp(16)), max(0, sz.Y-navH-size-gtx.Dp(16)))).Push(gtx.Ops)
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size, size))
		c := &u.sidebar.newChat
		ic := icNewChat
		if u.page == pageStatus {
			c = &u.status.add
			ic = icAddPhoto
		}
		clickable(g, c, func(gtx C) D {
			fillRRect(gtx, image.Rectangle{Max: gtx.Constraints.Max}, gtx.Dp(16), u.pal.Green)
			return centerIn(gtx, size, iconW(ic, 27, u.pal.OnGreen))
		})
		t.Pop()
	}
	if navH > 0 {
		nav := gtx
		nav.Constraints = layout.Exact(image.Pt(sz.X, navH))
		t := op.Offset(image.Pt(0, sz.Y-navH)).Push(gtx.Ops)
		u.layoutMobileNavigation(nav)
		t.Pop()
	}
	return D{Size: sz}
}
func (u *UI) layoutMobileNavigation(gtx C) D {
	p := u.pal
	fillRect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, max(1, gtx.Dp(1))), p.Divider)
	var children []layout.FlexChild
	for i, tab := range mobileTabs() {
		children = append(children, layout.Flexed(1, func(gtx C) D {
			c := u.btn("mobile:tab:" + itoa(i))
			if c.Clicked(gtx) {
				u.setPage(tab.page)
				u.setListHidden(false)
			}
			active := u.page == tab.page || tab.page == pageStatus && u.page == pageChannels
			return clickable(gtx, c, func(gtx C) D {
				sz := gtx.Constraints.Max
				gtx.Constraints = layout.Exact(sz)
				return layout.Center.Layout(gtx, func(gtx C) D {
					col := p.IconStrong
					if active {
						col = p.Green
					}
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							w, h := gtx.Dp(60), gtx.Dp(30)
							gtx.Constraints = layout.Exact(image.Pt(w, h))
							if active {
								fillRRect(gtx, image.Rect(0, 0, w, h), h/2, p.ChipActive)
							}
							return layout.Center.Layout(gtx, func(gtx C) D {
								switch tab.page {
								case pageChats:
									return chatsOutline(gtx, 25, col)
								case pageStatus:
									return statusIcon(gtx, 25, col, active)
								case pageCommunities:
									return drawIcon(gtx, icGroupsLine, 25, col)
								default:
									return drawIcon(gtx, icCall, 25, col)
								}
							})
						}), layout.Rigid(layout.Spacer{Height: 5}.Layout),
						layout.Rigid(u.label(12, u.locale.Text(tab.label), col, labelOpts{weight: font.Medium, maxLines: 2, align: text.Middle}).Layout))
				})
			})
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}
func (u *UI) mobileHasBack() bool {
	return u.voiceRecord.phase != recordIdle || u.acctMenu.open || u.menu.open || u.ctx.isOpen() || u.dialog.isOpen() || u.status.text.isOpen() || u.picker.open || u.viewer.open || u.gallery.open || u.info.open || u.search.open || u.msgInfo.open || u.newChat.open() || u.conv.selecting || u.conv.edit.msg != nil || u.conv.reply != nil || u.status.viewer.isOpen() || len(u.attach.files) > 0 || u.selected != nil || u.settings.detail != 0 || u.page != pageChats
}
func (u *UI) mobileBack() bool {
	if !u.mobileHasBack() {
		return false
	}
	channel := u.selected != nil && isChannelID(u.selected.ID)
	if u.selected == nil && !u.info.open && !u.search.open && !u.msgInfo.open && !u.dialog.isOpen() && !u.menu.open && !u.ctx.isOpen() && !u.status.viewer.isOpen() && !u.status.text.isOpen() && !u.viewer.open && !u.gallery.open && !u.newChat.open() && !u.acctMenu.open && !u.picker.open && len(u.attach.files) == 0 && u.settings.detail == 0 {
		u.setPage(pageChats)
		return true
	}
	u.escape()
	if channel && u.selected == nil {
		u.setPage(pageStatus)
	}
	return true
}
