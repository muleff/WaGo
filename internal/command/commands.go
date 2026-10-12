package command

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/skidy89/openWA/internal/model"
	"github.com/skidy89/openWA/internal/sticker"
)

// All lists the commands, in the order the picker shows them.
var All = []*Command{
	{
		Name: "add", Description: "command.add", Group: true, Admin: true,
		Options: []Option{{Name: "contact", Description: "command.add.contact",
			Kind: Contact, Required: true, Multiple: true}},
		Run: func(c *Context) error { return changeMembers(c, model.GroupAdd, "contact") },
	},
	{
		Name: "kick", Description: "command.kick", Group: true, Admin: true,
		Options: []Option{{Name: "member", Description: "command.kick.member", Kind: Member, Required: true, Multiple: true}},
		Run:     func(c *Context) error { return changeMembers(c, model.GroupRemove, "member") },
	},
	{
		Name: "promote", Description: "command.promote", Group: true, Admin: true,
		Options: []Option{{Name: "member", Description: "command.promote.member", Kind: Member, Required: true, Multiple: true,
			Filter: func(m model.Member) bool { return !m.Me && !m.Admin }}},
		Run: func(c *Context) error { return changeMembers(c, model.GroupPromote, "member") },
	},
	{
		Name: "demote", Description: "command.demote", Group: true, Admin: true,
		Options: []Option{{Name: "member", Description: "command.demote.member", Kind: Member, Required: true, Multiple: true,
			Filter: func(m model.Member) bool { return !m.Me && m.Admin }}},
		Run: func(c *Context) error { return changeMembers(c, model.GroupDemote, "member") },
	},
	{
		Name: "link", Description: "command.link", Group: true, Admin: true,
		Run: runLink,
	},
	{
		Name: "lockdown", Description: "command.lockdown", Group: true, Admin: true,
		Options: []Option{{Name: "mode", Description: "command.lockdown.mode",
			Kind: Choice, Choices: []string{"on", "off"}}},
		Run: runLockdown,
	},
	{
		Name: "description", Description: "command.description", Group: true, Admin: true,
		Options: []Option{{Name: "text", Description: "command.description.text", Kind: Text, Required: true}},
		Run:     runDescription,
	},
	{
		Name: "sticker", Description: "command.sticker",
		Options: []Option{
			{Name: "Packname", Description: "command.sticker.top", Kind: Text, Until: "#"},
			{Name: "Author", Description: "command.sticker.bottom", Kind: Text, Until: "#"},
			{Name: "Ai", Description: "command.sticker.ai", Kind: Choice, Choices: []string{"yes", "no"}},
			{Name: "Premium", Description: "command.sticker.premium", Kind: Choice, Choices: []string{"yes", "no"}},
		},
		Run: runSticker,
	},
	{
		Name: "purge", Description: "command.purge",
		Options: []Option{{Name: "count", Description: "command.purge.count", Kind: Number,
			Required: true, Min: 1, Max: maxPurge}},
		Run: runPurge,
	},
	{
		Name: "raffle", Description: "command.raffle", Group: true,
		Options: []Option{{Name: "winners", Description: "command.raffle.winners", Kind: Number, Min: 1, Max: 50}},
		Run:     runRaffle,
	},
	{
		Name: "calc", Description: "command.calc",
		Options: []Option{{Name: "sum", Description: "command.calc.sum",
			Kind: Text, Required: true}},
		Run:     runCalc,
		Preview: previewCalc,
	},
	{
		Name: "schedule", Description: "command.schedule",
		Options: []Option{
			{Name: "when", Description: "command.schedule.when", Kind: When,
				Required: true, Choices: []string{"30m", "1h", "3h", "tomorrow 08:00"}},
			{Name: "message", Description: "command.schedule.message", Kind: Text,
				Required: true, Mentions: true},
		},
		Run:     runSchedule,
		Preview: previewSchedule,
	},
	{
		Name: "scheduled", Description: "command.scheduled",
		Run: runScheduled,
	},
	{
		Name: "afk", Description: "command.afk",
		Options: []Option{{Name: "reason", Description: "command.afk.reason", Kind: Text}},
		Run:     runAFK,
	},
	{
		Name: "ghost", Description: "command.ghost",
		Gray: true, Run: runGhost,
	},
	{
		Name: "snippet", Description: "command.snippet",
		Options: []Option{
			{Name: "action", Description: "command.snippet.action",
				Kind: Choice, Choices: []string{"send", "save"}, Required: true},
			{Name: "snippet", Description: "command.snippet.snippet",
				Kind: Snippet},
		},
		Run: runSnippet,
	},
	{
		Name: "catch", Description: "command.catch",
		Options: []Option{{Name: "adn", Description: "Include available additionalNodes metadata", Kind: Choice, Choices: []string{"--adn"}}},
		Run:     runCatch,
	},
	{
		Name: "fakemsg", Description: "command.fakemsg",
		Gray: true,
		Options: []Option{
			{
				Name:        "sender",
				Description: "command.fakemsg.sender",
				Kind:        Text,
				Required:    true,
				Until:       "|",
			},
			{
				Name:        "text",
				Description: "command.fakemsg.text",
				Kind:        Text,
				Required:    true,
				Until:       "|",
			},
			{
				Name:        "quoted",
				Description: "command.fakemsg.quoted",
				Kind:        Text,
				Required:    true,
			},
		},
		Run: runFakeMsg,
	},
	{
		Name: "hidetag", Description: "command.hidetag",
		Group: true, Admin: true,
		Options: []Option{{Name: "text", Description: "command.hidetag.text", Kind: Text, Required: true}},
		Run:     runHideTag,
	},
	{Name: "replicate", Description: "command.replicate", Gray: true, Options: []Option{{Name: "text", Description: "command.replicate.text", Kind: Text}}, Run: runReplicateMessage},
}

// busy shows that the command is working, and returns its note.
func busy(c *Context, text string) *Note {
	n := &Note{Title: c.Input, Text: text, Busy: true}
	c.Note(n)
	return n
}

// fail turns a busy note into an error.
func fail(n *Note, text string) {
	n.Busy, n.Failed, n.Text = false, true, text
}

// changeMembers adds, removes, promotes or demotes the people in option
// opt, then says what happened to each.
func changeMembers(c *Context, action model.GroupAction, opt string) error {
	ids := c.IDs(opt)
	if len(ids) == 0 {
		return errors.New(c.T("Pick someone first."))
	}
	verb := map[model.GroupAction]string{model.GroupAdd: "Adding", model.GroupRemove: "Removing",
		model.GroupPromote: "Promoting", model.GroupDemote: "Dismissing"}[action]
	n := busy(c, c.T(verb)+"…")
	chat := c.Chat.ID
	c.Group(model.GroupRequest{ChatID: chat, Action: action, Members: ids}, func(ev model.GroupEvent) {
		n.Busy = false
		if ev.Err != "" {
			fail(n, ev.Err)
			return
		}
		var done []string
		var doneIDs []string
		var lines []string
		for _, r := range ev.Members {
			if r.Err == "" {
				done = append(done, r.Name)
				doneIDs = append(doneIDs, r.ID)
				continue
			}
			lines = append(lines, r.Name+" "+r.Err+".")
			if r.Invite != nil {
				n.Buttons = append(n.Buttons, inviteButton(c, n, chat, r))
			}
		}
		if len(done) > 0 {
			var s string
			switch action {
			case model.GroupAdd:
				s = fmt.Sprintf(c.T("Added %s to the group."), names(c.Locale, done))
			case model.GroupRemove:
				s = fmt.Sprintf(c.T("Removed %s from the group."), names(c.Locale, done))
				n.Buttons = append(n.Buttons, Button{Label: c.T("Add back"), Run: func() {
					addBack(c, n, chat, doneIDs)
				}})
			case model.GroupPromote:
				s = fmt.Sprintf(c.T(map[bool]string{true: "%s is now a group admin.", false: "%s are now group admins."}[len(done) == 1]), names(c.Locale, done))
			case model.GroupDemote:
				s = fmt.Sprintf(c.T(map[bool]string{true: "%s is no longer a group admin.", false: "%s are no longer group admins."}[len(done) == 1]), names(c.Locale, done))
			}
			lines = append([]string{s}, lines...)
		}
		n.Failed = len(done) == 0
		n.Text = strings.Join(lines, "\n")
		if n.Text == "" {
			n.Text = c.T("Nothing changed.")
		}
	})
	return nil
}

// addBack adds people removed by /kick back to the group.
func addBack(c *Context, n *Note, chat string, ids []string) {
	n.Buttons, n.Busy = nil, true
	c.Group(model.GroupRequest{ChatID: chat, Action: model.GroupAdd, Members: ids}, func(ev model.GroupEvent) {
		n.Busy = false
		if ev.Err != "" {
			n.Text += "\n" + ev.Err
			return
		}
		for _, r := range ev.Members {
			if r.Err == "" {
				n.Text += "\n" + fmt.Sprintf(c.T("Added %s back."), r.Name)
			} else {
				n.Text += "\n" + r.Name + " " + r.Err + "."
			}
		}
	})
}

// inviteButton sends someone whose privacy settings refused /add an
// invite to join, in your chat with them.
func inviteButton(c *Context, n *Note, chat string, r model.MemberResult) Button {
	b := Button{Label: fmt.Sprintf(c.T("Invite %s"), firstName(r.Name))}
	b.Run = func() {
		// The button goes; the note says how it went.
		for i := range n.Buttons {
			if n.Buttons[i].Label == b.Label {
				n.Buttons = append(n.Buttons[:i:i], n.Buttons[i+1:]...)
				break
			}
		}
		c.Group(model.GroupRequest{ChatID: chat, Action: model.GroupSendInvite, Members: []string{r.ID}, Invite: r.Invite},
			func(ev model.GroupEvent) {
				if ev.Err != "" {
					n.Text += "\n" + fmt.Sprintf(c.T("Couldn't invite %s: %s"), r.Name, ev.Err)
					return
				}
				n.Text += "\n" + fmt.Sprintf(c.T("Sent %s an invite to join."), r.Name)
			})
	}
	return b
}

func runLink(c *Context) error {
	n := busy(c, c.T("Getting the invite link…"))
	chat := c.Chat.ID
	var show func(ev model.GroupEvent)
	show = func(ev model.GroupEvent) {
		n.Busy = false
		if ev.Err != "" {
			fail(n, ev.Err)
			return
		}
		link := ev.Link
		n.Text = link
		n.Buttons = []Button{
			{Label: c.T("Copy link"), Run: func() { c.Copy(link) }},
			{Label: c.T("Reset link"), Danger: true, Run: func() {
				c.Confirm(c.T("Reset the invite link?"), c.T("The current link stops working. Anyone with it can't join anymore."),
					c.T("Reset link"), true, func() {
						n.Busy, n.Buttons = true, nil
						c.Group(model.GroupRequest{ChatID: chat, Action: model.GroupLink, On: true}, show)
					})
			}},
		}
	}
	c.Group(model.GroupRequest{ChatID: chat, Action: model.GroupLink}, show)
	return nil
}

func runLockdown(c *Context) error {
	on := true
	switch {
	case c.Text("mode") != "":
		on = c.Text("mode") == "on"
	case c.Info != nil:
		on = !c.Info.Announce // switch it
	}
	n := busy(c, c.T(map[bool]string{true: "Locking the group…", false: "Unlocking the group…"}[on]))
	c.Group(model.GroupRequest{ChatID: c.Chat.ID, Action: model.GroupAnnounce, On: on}, func(ev model.GroupEvent) {
		n.Busy = false
		switch {
		case ev.Err != "":
			fail(n, ev.Err)
		case on:
			n.Text = c.T("Only admins can send messages now. /lockdown off lets everyone send again.")
		default:
			n.Text = c.T("Everyone can send messages again.")
		}
	})
	return nil
}

func runDescription(c *Context) error {
	text := c.Text("text")
	n := busy(c, c.T("Changing the description…"))
	c.Group(model.GroupRequest{ChatID: c.Chat.ID, Action: model.GroupDescription, Text: text}, func(ev model.GroupEvent) {
		n.Busy = false
		if ev.Err != "" {
			fail(n, ev.Err)
			return
		}
		n.Text = c.T("Changed the group description.")
	})
	return nil
}

func runSticker(c *Context) error {
	chat := c.Chat.ID
	var packn, author string
	if c.Text("Packname") != "" {
		packn = c.Text("Packname")
	} else {
		packn = "OpenWA"
	}
	if c.Text("Author") != "" {
		author = c.Text("Author")
	} else {
		author = "cli"
	}
	pack := sticker.Pack{Name: packn, Author: author, AI_STICKER: c.Text("Ai") == "yes", PREMIUM: c.Text("Premium") == "yes"}

	makeSticker := func(read func() ([]byte, error)) {
		n := busy(c, c.T("Making a sticker…"))
		c.Do(func() func() {
			data, err := read()
			var webp []byte
			if err == nil {
				webp, err = sticker.FromImage(data, &pack)
			}
			return func() {
				switch {
				case errors.Is(err, sticker.ErrAnimated):
					fail(n, c.T("Text can't go on an animated sticker yet."))
					return
				case errors.Is(err, sticker.ErrTooLarge):
					fail(n, c.T("That picture is too big to make a sticker of."))
					return
				case err != nil:
					fail(n, c.T("Couldn't make a sticker of that picture."))
					return
				}
				c.Dismiss(n)
				if m := c.Backend.SendNewSticker(chat, webp, nil); m != nil {
					c.Sent(m)
				}
			}
		})
	}
	src := c.Reply
	switch {
	case src == nil:
		c.PickImage(func(path string) {
			if path != "" {
				makeSticker(func() ([]byte, error) { return os.ReadFile(path) })
			}
		})
	case src.Kind == model.KindSticker:
		// It's a sticker already: send it as it is.
		c.Backend.SendSticker(chat, src, nil)
	case src.Kind == model.KindImage && (src.Media == model.MediaImage || src.Media == model.MediaVideo):
		data := c.Backend.MediaData(src.ChatID, src.ID)
		if data == nil {
			return errors.New(c.T("It hasn't downloaded yet. Try again in a moment."))
		}
		makeSticker(func() ([]byte, error) { return data, nil })
	default:
		return errors.New(c.T("Reply to a photo or a sticker, or run /sticker without replying to pick a picture."))
	}
	return nil
}

// maxPurge is the most messages /purge deletes at once.
const maxPurge = 100

// revokeWindow is how long after sending a message it can still be
// deleted for everyone (WhatsApp allows about two days).
const revokeWindow = 60 * time.Hour

// runPurge deletes the last count messages for everyone, or, replying to
// a message, that one and the ones before it. Messages already deleted
// don't count. Those you can't delete for everyone (others' outside
// groups you administer, or too old) are left as they are.
func runPurge(c *Context) error {
	chat := c.Chat.ID
	count := c.Int("count", 1)
	msgs := purgeable(c.Backend, chat, c.Reply, count)
	if len(msgs) == 0 {
		return errors.New(c.T("There are no messages to delete."))
	}
	admin := c.Chat.IsGroup && c.Info != nil && isAdmin(c.Info)
	var del []*model.Message
	others, old, unsent := 0, 0, 0
	for _, m := range msgs {
		switch {
		case !m.FromMe && !admin:
			others++
		case m.FromMe && (m.Receipt == model.Pending || m.Receipt == model.Failed):
			unsent++
		case c.Now.Sub(m.Time) > revokeWindow:
			old++
		default:
			del = append(del, m)
		}
	}
	var skipped []string
	if others > 0 {
		s := fmt.Sprintf(c.T("%d from other people"), others)
		if c.Chat.IsGroup {
			s += " " + c.T("(only group admins can delete those)")
		}
		skipped = append(skipped, s)
	}
	if old > 0 {
		skipped = append(skipped, fmt.Sprintf(c.T("%d too old to delete for everyone"), old))
	}
	if unsent > 0 {
		skipped = append(skipped, fmt.Sprintf(c.T("%d not sent yet"), unsent))
	}
	left := strings.Join(skipped, ", ")
	if len(del) == 0 {
		return errors.New(fmt.Sprintf(c.T("None of them can be deleted: %s."), left))
	}
	body := ""
	if left != "" {
		body = fmt.Sprintf(c.T("Left as they are: %s."), left)
	}
	c.Confirm(fmt.Sprintf(c.T("Delete %s for everyone?"), plural(c.Locale, len(del), "message")), body, c.T("Delete for everyone"), true, func() {
		for _, m := range del {
			c.Backend.Delete(m, true)
		}
		text := fmt.Sprintf(c.T("Deleted %s for everyone."), plural(c.Locale, len(del), "message"))
		if left != "" {
			text += "\n" + fmt.Sprintf(c.T("Left as they are: %s."), left)
		}
		c.Note(&Note{Title: c.Input, Text: text})
	})
	return nil
}

// purgeable returns up to count messages of a chat that aren't deleted
// yet, newest first: the newest ones, or from reply back.
func purgeable(b model.Backend, chat string, reply *model.Message, count int) []*model.Message {
	var out []*model.Message
	var page []*model.Message
	if reply != nil {
		page = b.MessagesFrom(chat, reply.ID, 1)
		if len(page) == 0 {
			return nil
		}
	} else {
		page = b.Messages(chat, count)
	}
	for len(page) > 0 {
		for i := len(page) - 1; i >= 0 && len(out) < count; i-- {
			if page[i].Kind != model.KindDeleted {
				out = append(out, page[i])
			}
		}
		if len(out) == count {
			break
		}
		page = b.MessagesBefore(chat, page[0].ID, count)
	}
	return out
}

// runRaffle draws winners from the group's members, you left out, and
// sends them as a message that mentions them.
func runRaffle(c *Context) error {
	if c.Info == nil {
		return errors.New(c.T("The group's members haven't loaded yet. Try again in a moment."))
	}
	var pool []model.Member
	for _, m := range c.Info.Members {
		if !m.Me && m.ID != "" {
			pool = append(pool, m)
		}
	}
	n := c.Int("winners", 1)
	if len(pool) == 0 {
		return errors.New(c.T("There's no one else in the group to draw."))
	}
	if n > len(pool) {
		return errors.New(fmt.Sprintf(c.T("The group has only %s to draw from."), plural(c.Locale, len(pool), "other member")))
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	text, ids := raffleTextLocalized(c.Locale, raffleLines[rand.IntN(len(raffleLines))], pool[:n], len(pool))
	if m := c.Backend.Send(c.Chat.ID, model.Draft{Text: text, Mentions: ids}); m != nil {
		c.Sent(m)
	}
	return nil
}

// raffleText is the message announcing a raffle's winners, drawn from
// pool members, under line (one of raffleLines), and the IDs it mentions.
func raffleText(line string, winners []model.Member, pool int) (string, []string) {
	return raffleTextLocalized(nil, line, winners, pool)
}

func raffleTextLocalized(t Translator, line string, winners []model.Member, pool int) (string, []string) {
	var b strings.Builder

	b.WriteString(translate(t, "🎲 *Raffle*"))
	b.WriteByte('\n')
	b.WriteString(line)
	b.WriteByte(':')

	ids := make([]string, len(winners))

	for i, w := range winners {
		ids[i] = w.ID

		user, _, _ := strings.Cut(w.ID, "@")

		b.WriteByte('\n')
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". @")
		b.WriteString(user)
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(translate(t, "_Drawn at random from %s._"), plural(t, pool, "member")))

	return b.String(), ids
}

// lastAnswer is the answer of the last /calc, which "ans" stands for. Only
// the UI goroutine runs commands.
var lastAnswer float64

// runCalc sends the sum with its answer: "12 x 4500 = 54 000".
func runCalc(c *Context) error {
	sum := strings.TrimSpace(c.Text("sum"))
	v, err := Calc(sum, lastAnswer)
	if err != nil {
		return errors.New(c.T("Couldn't work that out: ") + err.Error() + ".")
	}
	if m := c.Backend.Send(c.Chat.ID, model.Draft{Text: sum + " = " + FormatNumber(v), Reply: c.Reply}); m != nil {
		lastAnswer = v
		c.Sent(m)
	}
	return nil
}

// previewCalc shows the answer while the sum is typed. A sum that only
// isn't finished yet shows nothing.
func previewCalc(in *Input) (string, bool) {
	sum := strings.TrimSpace(in.Text("sum"))
	if sum == "" {
		return "", false
	}
	v, err := Calc(sum, lastAnswer)
	switch {
	case errors.Is(err, errUnfinished):
		return "", false
	case err != nil:
		return strings.ToUpper(err.Error()[:1]) + err.Error()[1:], false
	}
	return "= " + FormatNumber(v), true
}

// plural is "1 message" or "3 messages".
func plural(t Translator, n int, noun string) string {
	if n == 1 {
		return "1 " + translate(t, noun)
	}
	return strconv.Itoa(n) + " " + translate(t, noun+"s")
}

// names lists names as a sentence: "A", "A and B", "A, B and C".
func names(t Translator, ns []string) string {
	switch len(ns) {
	case 0:
		return ""
	case 1:
		return ns[0]
	}
	return strings.Join(ns[:len(ns)-1], ", ") + " " + translate(t, "and") + " " + ns[len(ns)-1]
}

func translate(t Translator, text string) string {
	if t == nil {
		return text
	}
	return t.Text(text)
}

func firstName(s string) string {
	s = strings.TrimPrefix(s, "~")
	if i := strings.IndexByte(s, ' '); i > 0 && !strings.HasPrefix(s, "+") {
		return s[:i]
	}
	return s
}
