package ui

import (
	"github.com/muleff/WaGo/internal/model"
	"strings"
	"time"
)

func (u *UI) clockText(t time.Time) string {
	t = t.In(time.Local)
	if !u.mobile {
		return t.Format("15:04")
	}
	suffix := "a. m."
	if t.Hour() >= 12 {
		suffix = "p. m."
	}
	if u.locale.Language() != "es" {
		return t.Format("3:04 PM")
	}
	return t.Format("3:04") + " " + suffix
}

// Presence is backend-generated copy, unlike contact names and messages.
func (u *UI) presenceText(value string) string {
	if value == "online" {
		return u.locale.Text(value)
	}
	tail, ok := strings.CutPrefix(value, "last seen ")
	if !ok {
		return value
	}
	clock := func(v string) string {
		if t, e := time.ParseInLocation("15:04", v, time.Local); e == nil {
			return u.clockText(t)
		}
		return v
	}
	if v, ok := strings.CutPrefix(tail, "today at "); ok {
		tail = u.locale.Textf("today at %s", clock(v))
	} else if v, ok := strings.CutPrefix(tail, "yesterday at "); ok {
		tail = u.locale.Textf("yesterday at %s", clock(v))
	} else if day, at, ok := strings.Cut(tail, " at "); ok {
		tail = u.locale.Textf("%s at %s", day, clock(at))
	}
	return u.locale.Textf("last seen %s", tail)
}

func (u *UI) typingText(c *model.Chat) string {
	if len(c.Typing) == 0 {
		return ""
	}
	if !c.IsGroup {
		return u.locale.Text("typing…")
	}
	name := func(i int) string { return shortName(c.Typing[i].Name) }
	switch len(c.Typing) {
	case 1:
		return u.locale.Textf("%s is typing…", name(0))
	case 2:
		return u.locale.Textf("%s and %s are typing…", name(0), name(1))
	case 3:
		return u.locale.Textf("%s, %s and %s are typing…", name(0), name(1), name(2))
	default:
		return u.locale.Textf("%s, %s and %d others are typing…", name(0), name(1), len(c.Typing)-2)
	}
}

func (u *UI) followerText(n int) string {
	v := followers(n)
	if v == "Channel" {
		return u.locale.Text(v)
	}
	v = strings.ReplaceAll(v, " followers", " "+u.locale.Text("followers"))
	return strings.ReplaceAll(v, " follower", " "+u.locale.Text("follower"))
}

func (u *UI) mediaText(m *model.Message) string {
	if m.Kind != model.KindViewOnce && m.Text != "" && m.Media != model.MediaVoice && m.Media != model.MediaAudio {
		return m.Text
	}
	return u.locale.Text(mediaLabel(m))
}
