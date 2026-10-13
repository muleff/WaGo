package ui

import (
	"gioui.org/layout"
	"image"
)

func (u *UI) layoutMobileComposerControls(gtx C, sv float32) D {
	p := u.pal
	c := &u.conv
	return layout.Flex{Alignment: layout.End}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return vcenter(gtx, gtx.Dp(50), func(gtx C) D {
				return layout.Inset{Left: 2, Right: 2}.Layout(gtx, func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return u.iconButton(gtx, &c.emoji, icEmoji, 40, 25, p.IconStrong) }),
						layout.Flexed(1, func(gtx C) D {
							if c.editorElsewhere {
								txt := u.locale.Text("Adding a caption")
								if u.dialog.kind == dialogPoll {
									txt = u.locale.Text("Creating a poll")
								}
								return u.label(15, txt, p.TextSecondary).Layout(gtx)
							}
							return u.layoutComposerEditor(gtx, u.locale.Text("Message"))
						}),
						layout.Rigid(func(gtx C) D { return u.iconButton(gtx, &c.attach, icAttach, 40, 24, p.IconStrong) }),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: 8}.Layout),
		layout.Rigid(func(gtx C) D {
			return clickable(gtx, &c.send, func(gtx C) D {
				size := gtx.Dp(48)
				fillCircle(gtx, image.Pt(size/2, size/2), size/2, p.Green)
				ic := icMic
				if sv > 0.5 {
					ic = icSend
					if c.edit.msg != nil {
						ic = icTick
					}
				}
				return centerIn(gtx, size, iconW(ic, 25, p.OnGreen))
			})
		}),
	)
}
