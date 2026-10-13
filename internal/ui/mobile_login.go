package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"github.com/muleff/WaGo/internal/model"
)

func (u *UI) layoutMobileLogin(gtx C) D {
	p := u.pal
	// Android's keyboard can reduce the usable height below the card's
	// natural height. A scrollable form keeps Next and the code reachable.
	return u.mobileLoginList.Layout(gtx, 1, func(gtx C, _ int) D {
		return layout.Inset{Left: 12, Right: 12, Top: 16, Bottom: 16}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(480))
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(u.layoutMobileLoginCard),
				layout.Rigid(layout.Spacer{Height: 16}.Layout),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return drawIcon(gtx, icLock, 14, p.TextSecondary) }),
						layout.Rigid(layout.Spacer{Width: 5}.Layout),
						layout.Rigid(u.label(13, u.locale.Text("PolicyE2E"), p.TextSecondary).Layout),
					)
				}),
			)
		})
	})
}

func (u *UI) layoutMobileLoginCard(gtx C) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X

	m := op.Record(gtx.Ops)
	dims := layout.Inset{
		Left: 0, Right: 0,
		Top: 16, Bottom: 16,
	}.Layout(gtx, func(gtx C) D {
		if !u.verifyFading {
			return u.layoutMobileLoginForm(gtx, u.verifyStep)
		}

		v := easeOut(u.verifyAnim.step(gtx, true, durDialog))
		var oldDims, newDims D
		old, pass := fadeOut(gtx)
		withOpacity(old, 1-v, func() { oldDims = u.layoutMobileLoginForm(old, u.verifyFrom) })
		pass()
		withOpacity(gtx, v, func() { newDims = u.layoutMobileLoginForm(gtx, u.verifyStep) })
		if v >= 1 {
			u.verifyFading = false
			u.verifyFrom = u.verifyStep
		}
		return D{Size: image.Pt(
			max(oldDims.Size.X, newDims.Size.X),
			max(oldDims.Size.Y, newDims.Size.Y),
		)}
	})

	call := m.Stop()

	call.Add(gtx.Ops)
	return dims
}

func (u *UI) layoutMobileLoginForm(gtx C, verify bool) D {
	if verify {
		return u.layoutMobileVerifyForm(gtx)
	}
	return u.layoutMobilePhoneForm(gtx)
}

func (u *UI) layoutMobilePhoneForm(gtx C) D {
	p := u.pal
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			sz := gtx.Dp(48)
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			fillCircle(gtx, image.Pt(sz/2, sz/2), sz/2, p.Green)
			return layout.Center.Layout(gtx, func(gtx C) D {
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(22), gtx.Dp(22)))
				return drawIcon(gtx, icLink, 22, p.OnGreen)
			})
		}),
		layout.Rigid(layout.Spacer{Height: 20}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.label(
				24,
				u.locale.Text("Link your WhatsApp"),
				p.Text,
				labelOpts{
					weight:   font.Medium,
					maxLines: 1,
				},
			).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 5}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.label(
				14,
				u.locale.Text("Enter the phone number connected to your WhatsApp account. We'll use it to prepare this device for linking."),
				p.TextSecondary,
				labelOpts{maxLines: 4},
			).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 22}.Layout),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					txt := u.locale.Text("Country code")
					col := p.TextSecondary
					return vcenter(gtx, gtx.Constraints.Min.Y, u.label(12, txt, col, labelOpts{maxLines: 1}).Layout)
				}),
				layout.Rigid(layout.Spacer{Width: 12}.Layout),
				layout.Flexed(1, func(gtx C) D {
					return u.label(12, u.locale.Text("Phone number"), p.TextSecondary, labelOpts{maxLines: 1}).Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: 7}.Layout),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Dp(48)
			gtx.Constraints.Max.Y = gtx.Dp(48)
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Dp(76)
					gtx.Constraints.Max.X = gtx.Dp(76)
					return u.layoutCountryCode(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: 12}.Layout),
				layout.Flexed(1, func(gtx C) D {
					return u.layoutPhoneNumber(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: 20}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.label(
				13,
				u.locale.Text("After you continue, follow the instructions in WhatsApp on your primary phone."),
				p.TextSecondary,
				labelOpts{maxLines: 3},
			).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 20}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.layoutPairingButton(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if u.mobile {
				return D{}
			}
			return layout.Inset{Top: 12}.Layout(gtx, func(gtx C) D {
				return u.dialogButton(gtx, "login:mobile-pair-back", dialogButton{
					label: u.locale.Text("Back to QR code"),
				})
			})
		}),
		layout.Rigid(layout.Spacer{Height: 20}.Layout),
		layout.Rigid(func(gtx C) D {
			if u.conn.State == model.StateError && u.conn.Err != "" {
				return u.label(
					14,
					u.conn.Err,
					p.Danger,
					labelOpts{maxLines: 3},
				).Layout(gtx)
			}
			return D{}
		}),
	)
}

func (u *UI) layoutMobileVerifyForm(gtx C) D {
	p := u.pal
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			sz := gtx.Dp(48)
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			fillCircle(gtx, image.Pt(sz/2, sz/2), sz/2, p.Green)
			return layout.Center.Layout(gtx, func(gtx C) D {
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(22), gtx.Dp(22)))
				return drawIcon(gtx, icLock, 22, p.OnGreen)
			})
		}),
		layout.Rigid(layout.Spacer{Height: 18}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.label(24, u.locale.Text("Your 8-character code"), p.Text,
				labelOpts{weight: font.Medium, maxLines: 1}).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 10}.Layout),
		layout.Rigid(func(gtx C) D {
			phone := "+" + u.countryCode.Text() + " " + u.phoneNumber.Text()
			return u.label(14, u.locale.Textf("Enter this code in WhatsApp on your phone: %s", phone), p.TextSecondary,
				labelOpts{maxLines: 2}).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 28}.Layout),
		layout.Rigid(u.layoutMobileCodeBoxes),
		layout.Rigid(layout.Spacer{Height: 12}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.label(13, u.locale.Text("The code contains eight letters or numbers."), p.TextSecondary,
				labelOpts{maxLines: 1}).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 24}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.layoutResendCode(gtx, u.conn.State != model.StateQRExpired || u.conn.State == model.StateError)
		}),
		layout.Rigid(layout.Spacer{Height: 12}.Layout),
		layout.Rigid(func(gtx C) D {
			return u.dialogButton(gtx, "login:code-back", dialogButton{
				label: u.locale.Text("Change phone number"),
			})
		}),
	)
}
func (u *UI) layoutResendCode(gtx C, disabled bool) D {
	return u.dialogButton(gtx, "login:code-resend", dialogButton{
		label:    u.locale.Text("Resend code"),
		disabled: disabled,
	})
}

func (u *UI) layoutMobileCodeBoxes(gtx C) D {
	p := u.pal
	var codeChars []byte
	code := u.verifyCode.Text()

	for i := 0; i < len(code); i++ {
		c := code[i]
		if (c >= '0' && c <= '9') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') {
			codeChars = append(codeChars, c)
		}
	}

	if len(codeChars) > 8 {
		codeChars = codeChars[:8]
	}

	formatted := string(codeChars)
	if len(codeChars) > 4 {
		formatted = string(codeChars[:4]) + "-" + string(codeChars[4:])
	}

	h := gtx.Dp(52)

	return layout.Center.Layout(gtx, func(gtx C) D {
		w := gtx.Constraints.Max.X
		gtx.Constraints.Min.Y = h
		gtx.Constraints.Max.Y = h

		size := image.Pt(w, h)
		gtx.Constraints = layout.Exact(size)

		borderRRect(gtx, image.Rectangle{Max: size}, gtx.Dp(9), p.Panel, p.PanelBorder)

		if formatted != "" {
			label := u.label(23, formatted, p.Text, labelOpts{maxLines: 1})
			labelGtx := gtx
			labelGtx.Constraints.Min = image.Point{}
			labelGtx.Constraints.Max = size
			lbl := record(labelGtx, label.Layout)
			lbl.at(gtx, (w-lbl.size.X)/2, (h-lbl.size.Y)/2)
		}

		return D{Size: size}
	})
}

func (u *UI) layoutCountryCode(gtx C) D {
	p := u.pal

	u.countryCode.SingleLine = true
	u.countryCode.Filter = "0123456789"
	u.countryCode.InputHint = key.HintNumeric

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			borderRRect(gtx, image.Rectangle{Max: gtx.Constraints.Max}, gtx.Dp(10), p.Panel, p.PanelBorder)
			return D{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Inset{
				Top: 14, Bottom: 14,
				Left: 10, Right: 8,
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {

						return u.label(18, u.locale.Text("+"), p.Text).Layout(gtx)
					}),
					layout.Flexed(1, func(gtx C) D {
						return material.Editor(u.th, &u.countryCode, "52").Layout(gtx)
					}),
				)
			})
		}),
	)
}

func (u *UI) layoutPairingButton(gtx C) D {
	country, number := u.countryCode.Text(), u.phoneNumber.Text()
	disabled := u.login.pairing != nil || len(country) < 1 || len(country) > 3 || len(number) < 7 || len(country)+len(number) > 15
	label := u.locale.Text("Next")
	if u.login.pairing != nil {
		label = u.locale.Text("Connecting...")
	}
	return u.dialogButton(gtx, "login:pair", dialogButton{
		label:    label,
		primary:  true,
		disabled: disabled,
	})
}
func (u *UI) layoutPhoneNumber(gtx C) D {
	p := u.pal
	u.phoneNumber.SingleLine = true
	u.phoneNumber.Filter = "0123456789"
	u.phoneNumber.InputHint = key.HintTelephone
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			borderRRect(gtx, image.Rectangle{Max: gtx.Constraints.Max}, gtx.Dp(10), p.Panel, p.PanelBorder)
			return D{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Inset{Top: 14, Bottom: 14, Left: 12, Right: 12}.Layout(gtx, func(gtx C) D {
				return material.Editor(u.th, &u.phoneNumber, u.locale.Text("Phone number")).Layout(gtx)
			})
		}),
	)
}
