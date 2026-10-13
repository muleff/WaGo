package ui

import (
	"image"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/muleff/WaGo/internal/model"
)

type snippetSettings struct {
	items                           []model.Snippet
	search, name, body              widget.Editor
	query                           string
	id                              int64
	payload, editing, loading, busy bool
	message                         string
}

func (u *UI) openSnippetSettings(id int64) {
	u.setPage(pageSettings)
	u.openSettings(settingSnippets)
	if id != 0 {
		u.editSnippet(id)
	}
}

func (u *UI) loadSnippetSettings() {
	s := &snippetSettings{loading: true}
	s.search.SingleLine = true
	u.settings.snippets = s
	u.settings.stale = true
	b := u.backend
	u.slashDo(func() func() {
		items, err := b.Snippets()
		return func() {
			if u.settings.snippets != s {
				return
			}
			s.loading = false
			s.items = items
			if err != nil {
				s.message = err.Error()
			}
			u.settings.stale = true
		}
	})
}

func newSnippetEditor() *snippetSettings {
	s := &snippetSettings{editing: true}
	s.name.SingleLine = true
	s.name.MaxLen = 80
	s.body.MaxLen = model.MaxSnippetBytes
	return s
}

func (u *UI) editSnippet(id int64) {
	s := newSnippetEditor()
	u.settings.snippets = s
	u.settings.stale = true
	u.settings.detailList.Position = layout.Position{}
	if id == 0 {
		u.requestFocus(&s.name)
		return
	}
	s.loading = true
	b := u.backend
	u.slashDo(func() func() {
		item, err := b.Snippet(id)
		return func() {
			if u.settings.snippets != s {
				return
			}
			s.loading = false
			if err != nil {
				s.message = err.Error()
				s.editing = false
			} else {
				s.id = item.ID
				s.payload = item.Payload
				s.name.SetText(item.Name)
				s.body.SetText(item.Body)
			}
			u.settings.stale = true
		}
	})
}

func (u *UI) saveSnippetSettings() {
	s := u.settings.snippets
	if s == nil || s.busy || s.loading {
		return
	}
	item := model.Snippet{ID: s.id, Name: s.name.Text(), Payload: s.payload, Body: s.body.Text()}
	s.busy = true
	s.message = u.locale.Text("Saving…")
	u.settings.stale = true
	b := u.backend
	u.slashDo(func() func() {
		saved, err := b.SaveSnippet(item)
		return func() {
			if u.settings.snippets != s {
				return
			}
			s.busy = false
			if err != nil {
				s.message = err.Error()
			} else {
				s.id = saved.ID
				s.name.SetText(saved.Name)
				s.message = u.locale.Text("Saved.")
				u.snippetsChanged()
			}
			u.settings.stale = true
		}
	})
}

func (u *UI) deleteSnippetSettings() {
	s := u.settings.snippets
	if s == nil || s.id == 0 || s.busy {
		return
	}
	b, id := u.backend, s.id
	u.confirm(u.locale.Text("Delete snippet?"), u.locale.Text("Remove")+" "+s.name.Text()+" "+u.locale.Text("from your saved messages?"), dialogButton{label: u.locale.Text("Delete"), danger: true, run: func() {
		s.busy = true
		u.settings.stale = true
		u.slashDo(func() func() {
			err := b.DeleteSnippet(id)
			return func() {
				if u.settings.snippets != s {
					return
				}
				s.busy = false
				if err != nil {
					s.message = err.Error()
					u.settings.stale = true
				} else {
					u.snippetsChanged()
					u.loadSnippetSettings()
				}
			}
		})
	}})
}

func (u *UI) snippetSettingsRows() []settingsSection {
	s := u.settings.snippets
	if s == nil {
		return nil
	}
	if s.loading {
		return []settingsSection{{note: u.locale.Text("Loading snippets…")}}
	}
	if s.editing {
		field := func(key string, ed *widget.Editor, hint string, large bool) settingRow {
			return settingRow{key: key, kind: setCustom, w: func(gtx C) D {
				if s.busy {
					gtx = gtx.Disabled()
				}
				return layout.Inset{Left: 24, Right: 24, Bottom: 12}.Layout(gtx, func(gtx C) D {
					if large {
						gtx.Constraints.Min.Y = gtx.Dp(180)
						gtx.Constraints.Max.Y = gtx.Dp(240)
					}
					return u.pollLine(gtx, ed, hint, false)
				})
			}}
		}
		secs := []settingsSection{
			{title: u.locale.Text("Name"), rows: []settingRow{field("snippet:name", &s.name, u.locale.Text("Generated automatically when empty"), false)}},
			{title: u.locale.Text("Message type"), rows: []settingRow{
				{key: "snippet:text", kind: setRadio, title: u.locale.Text("Text"), on: !s.payload, run: func() {
					if !s.busy {
						s.payload = false
					}
				}},
				{key: "snippet:json", kind: setRadio, title: u.locale.Text("Payload JSON"), on: s.payload, run: func() {
					if !s.busy {
						s.payload = true
					}
				}},
			}},
			{title: u.locale.Text("Message"), rows: []settingRow{field("snippet:body", &s.body, u.locale.Text("Enter a message or paste message JSON"), true)}, note: u.snippetVarsNote()},
			{rows: []settingRow{{key: "snippet:save", ic: icTick, title: u.locale.Text("Save snippet"), run: u.saveSnippetSettings}}, note: s.message},
		}
		if s.id != 0 {
			secs = append(secs, settingsSection{rows: []settingRow{{key: "snippet:delete", ic: icDelete, title: u.locale.Text("Delete snippet"), danger: true, run: u.deleteSnippetSettings}}})
		}
		return secs
	}
	rows := []settingRow{
		{key: "snippet:search", kind: setCustom, w: func(gtx C) D {
			return layout.Inset{Left: 24, Right: 24, Bottom: 10}.Layout(gtx, func(gtx C) D {
				return u.searchField(gtx, &s.search, u.locale.Text("Search snippets"))
			})
		}},
		{key: "snippet:add", ic: icAdd, title: u.locale.Text("New snippet"), sub: u.locale.Text("Save text or a message payload"), run: func() { u.editSnippet(0) }},
	}
	q := strings.ToLower(strings.TrimSpace(s.query))
	for _, item := range s.items {
		if q != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Preview), q) {
			continue
		}
		kind := u.locale.Text("Text")
		if item.Payload {
			kind = u.locale.Text("Payload")
		}
		rows = append(rows, settingRow{key: "snippet:item:" + strconv.FormatInt(item.ID, 10), ic: icDocument, title: item.Name, sub: kind + " · " + item.Preview, trailing: icEdit, run: func() { u.editSnippet(item.ID) }})
	}
	note := s.message
	if len(s.items) == 0 && note == "" {
		note = u.locale.Text("No snippets yet. Add one here, or reply to a message and use /snippet save.")
	}
	return []settingsSection{{rows: rows, note: note}, {note: u.locale.Text("Type /snippet send and a snippet's name in a chat to send it. Snippets belong to this account.")}}
}

// snippetVarsNote lists the variables under the message editor.
func (u *UI) snippetVarsNote() string {
	var sb strings.Builder
	sb.WriteString(u.locale.Text("Variables, filled in when you send (in a payload, in its text and captions):"))
	for _, v := range model.SnippetVars {
		sb.WriteString("\n{")
		sb.WriteString(v.Name)
		sb.WriteString("}: ")
		sb.WriteString(u.locale.Text("snippet.var." + v.Name))
	}
	sb.WriteString("\n")
	sb.WriteString(u.locale.Text("Write \\{name} to send {name} as it is."))
	return sb.String()
}

// snippetDialog is /catch's payload of a message.
type snippetDialog struct {
	chat, id        string
	additionalNodes bool
	loading, busy   bool
	message         string
	body            widget.Editor
}

// snippetFixtures are the snippets cmd/screenshot shows.
var snippetFixtures = []model.Snippet{
	{ID: 1, Name: "Greeting", Preview: "{greeting} {first}! Thanks for reaching out. How can I help?"},
	{ID: 2, Name: "Office hours", Preview: "We're available Monday to Friday, 09:00–17:00."},
	{ID: 3, Name: "Payment details", Payload: true, Preview: "Here are the payment details for your order."},
}

// showSnippetPreview supplies fixtures to cmd/screenshot without writing
// sample snippets into a connected account's database.
func (u *UI) showSnippetPreview(name string) {
	body := "{\n  \"extendedTextMessage\": {\n    \"text\": \"Here are the payment details for your order.\"\n  }\n}"
	switch name {
	case "snippets", "snippetedit":
		u.ShowPage("snippets")
		s := &snippetSettings{items: snippetFixtures}
		s.search.SingleLine = true
		if name == "snippetedit" {
			s = newSnippetEditor()
			s.id = 3
			s.payload = true
			s.name.SetText("Payment details")
			s.body.SetText(body)
		}
		u.settings.snippets = s
		u.settings.stale = true
	case "snippet":
		// /snippet's picker over the composer.
		u.slash.on = true
		u.slash.snippets, u.slash.snippetsOK = snippetFixtures, true
		u.slash.snippetsVer++
		ed := &u.conv.composer
		ed.SetText("/snippet send ")
		ed.SetCaret(ed.Len(), ed.Len())
		u.requestFocus(ed)
	case "catch":
		d := &snippetDialog{}
		d.body.ReadOnly = true
		d.body.SetText(body)
		u.dialog = dialogState{kind: dialogPayload, snippet: d}
	}
}

// snippetsChanged drops the snippets /snippet offers, to load them again.
func (u *UI) snippetsChanged() {
	s := &u.slash
	s.snippets, s.snippetsOK, s.snippetsLoad = nil, false, nil
	s.snippetsVer++
}

// slashSnippets returns the saved snippets for /snippet's picker, and
// starts loading them the first time.
func (u *UI) slashSnippets() []model.Snippet {
	s := &u.slash
	if s.snippetsOK || s.snippetsLoad != nil {
		return s.snippets
	}
	load := new(bool)
	s.snippetsLoad = load
	b := u.backend
	u.slashDo(func() func() {
		items, _ := b.Snippets()
		return func() {
			if s.snippetsLoad != load {
				return // changed meanwhile
			}
			s.snippets, s.snippetsOK, s.snippetsLoad = items, true, nil
			s.snippetsVer++
		}
	})
	return nil
}

// snippetValues are the rows /snippet send offers for word: the snippets
// whose name has it, those starting with it first.
func (u *UI) snippetValues(word string) []slashValue {
	q := strings.ToLower(strings.TrimSpace(word))
	var first, rest []slashValue
	for _, item := range u.slashSnippets() {
		name := strings.ToLower(item.Name)
		v := slashValue{name: item.Name, sub: item.Preview, ic: icDocument}
		switch {
		case strings.HasPrefix(name, q):
			first = append(first, v)
		case strings.Contains(name, q) || strings.Contains(strings.ToLower(item.Preview), q):
			rest = append(rest, v)
		}
	}
	return append(first, rest...)
}

// snippetTyped reports whether v is a whole snippet's name, so that
// Enter sends it rather than picks.
func (u *UI) snippetTyped(v string) bool {
	v = strings.TrimSpace(v)
	for _, item := range u.slash.snippets {
		if strings.EqualFold(item.Name, v) {
			return true
		}
	}
	return false
}

func (u *UI) openPayload(chat, id string, additionalNodes bool) {
	d := &snippetDialog{chat: chat, id: id, additionalNodes: additionalNodes, loading: true}
	d.body.ReadOnly = true
	u.dialog = dialogState{kind: dialogPayload, snippet: d}
	b := u.backend
	u.slashDo(func() func() {
		var body string
		var err error
		if additionalNodes {
			body, err = b.MessagePayloadWithAdditionalNodes(chat, id)
		} else {
			body, err = b.MessagePayload(chat, id)
		}
		return func() {
			if u.dialog.snippet != d {
				return
			}
			d.loading = false
			if err != nil {
				d.message = err.Error()
			} else {
				d.body.SetText(body)
			}
		}
	})
}

// snippetVars fills a snippet's variables (model.SnippetVars) for chat:
// {name} is the contact, or in a group the author of reply (else the group).
func (u *UI) snippetVars(chat string, reply *model.Message) map[string]string {
	var chatName string
	group := false
	if c := u.chatByID(chat); c != nil {
		chatName, group = plainText(u.listName(c)), c.IsGroup
	}
	name, quote := chatName, ""
	if reply != nil {
		quote = plainText(reply.Text)
		if group && !reply.FromMe && reply.Sender != "" {
			name = plainText(reply.Sender)
		}
	}
	return model.SnippetValues(name, chatName, u.meName(), quote, u.now())
}

// snippetPanel is /catch's dialog: a message's payload as JSON.
func (u *UI) snippetPanel(gtx C) D {
	d := u.dialog.snippet
	if d == nil {
		return D{}
	}
	title := "Message payload"
	if d.additionalNodes {
		title += " + additionalNodes"
	}
	active := u.dialog.isOpen()
	if active && u.btn("snip:close").Clicked(gtx) {
		u.closeDialog()
	}
	if active && !d.loading && !d.busy {
		if u.btn("snip:copy").Clicked(gtx) && d.body.Len() > 0 {
			u.pendingCopy = d.body.Text()
			d.message = "JSON copied."
		}
		if u.btn("snip:export").Clicked(gtx) && d.body.Len() > 0 {
			d.busy = true
			b := u.backend
			u.slashDo(func() func() {
				var path string
				var err error
				if d.additionalNodes {
					path, err = b.ExportMessagePayloadWithAdditionalNodes(d.chat, d.id)
				} else {
					path, err = b.ExportMessagePayload(d.chat, d.id)
				}
				return func() {
					if u.dialog.snippet != d {
						return
					}
					d.busy = false
					if err != nil {
						d.message = err.Error()
					} else {
						d.message = "Saved to " + path
					}
				}
			})
		}
	}
	w := min(gtx.Dp(560), gtx.Constraints.Max.X-gtx.Dp(32))
	h := min(gtx.Dp(560), gtx.Constraints.Max.Y-gtx.Dp(48))
	gtx.Constraints = layout.Exact(image.Pt(w, h))
	return layout.UniformInset(20).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, u.label(19, title, u.pal.Text, labelOpts{weight: font.SemiBold, maxLines: 1}).Layout),
					layout.Rigid(func(gtx C) D { return u.iconButton(gtx, u.btn("snip:close"), icClose, 36, 22, u.pal.Icon) }))
			}),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Flexed(1, func(gtx C) D {
				if d.loading {
					return u.label(15, u.locale.Text("Loading…"), u.pal.TextSecondary).Layout(gtx)
				}
				ed := material.Editor(u.th, &d.body, "")
				ed.Color = u.pal.Text
				ed.TextSize = 13
				ed.Font.Typeface = monoTypeface
				return ed.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				if d.message == "" {
					return D{}
				}
				return layout.Inset{Top: 10, Bottom: 6}.Layout(gtx, u.label(13, d.message, u.pal.TextSecondary, labelOpts{maxLines: 3}).Layout)
			}),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(func(gtx C) D {
				button := func(key, label string, primary, disabled bool) layout.FlexChild {
					return layout.Rigid(func(gtx C) D {
						return u.dialogButton(gtx, key, dialogButton{label: label, primary: primary, disabled: disabled})
					})
				}
				return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
					button("snip:export", "Export JSON", false, d.loading || d.busy || d.body.Len() == 0),
					button("snip:copy", "Copy JSON", true, d.loading || d.body.Len() == 0))
			}),
		)
	})
}
