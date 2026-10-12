package command

import (
	"errors"
	"strings"

	"github.com/skidy89/openWA/internal/model"
)

func runSnippet(c *Context) error {
	if c.Chat == nil {
		return errors.New("Open a chat first.")
	}
	v := strings.TrimSpace(c.Text("snippet"))
	if c.Text("action") == "save" {
		return saveSnippet(c, v)
	}
	if v == "" {
		return errors.New("Pick a snippet to send.")
	}
	chat, reply := c.Chat.ID, c.Reply
	vars := c.SnippetVars(chat, reply)
	c.Do(func() func() {
		items, err := c.Backend.Snippets()
		return func() {
			if err != nil {
				fail(c.note(), err.Error())
				return
			}
			for _, s := range items {
				if strings.EqualFold(s.Name, v) {
					m, err := c.Backend.SendSnippet(chat, s.ID, reply, vars)
					if err != nil {
						fail(c.note(), err.Error())
					} else {
						c.Sent(m)
					}
					return
				}
			}
			fail(c.note(), "There's no snippet called "+v+". Add one in Settings > Snippets.")
		}
	})
	return nil
}

// note shows a new note for the command, to fail.
func (c *Context) note() *Note {
	n := &Note{Title: c.Input}
	c.Note(n)
	return n
}

func saveSnippet(c *Context, name string) error {
	if c.Reply == nil {
		return errors.New("Reply to a message, then use /snippet save.")
	}
	chat, id := c.Reply.ChatID, c.Reply.ID
	n := busy(c, "Saving snippet…")
	c.Do(func() func() {
		s, err := c.Backend.SaveMessageSnippet(chat, id, name)
		return func() {
			if err != nil {
				fail(n, err.Error())
				return
			}
			c.SnippetsChanged()
			n.Busy = false
			n.Text = "Saved as " + s.Name + ". Send it with /snippet send " + s.Name + "."
			n.Buttons = []Button{{Label: "Edit in Settings", Run: func() { c.EditSnippet(s.ID) }}}
		}
	})
	return nil
}

func runCatch(c *Context) error {
	if c.Reply == nil {
		return errors.New("Reply to a message, then use /catch.")
	}
	adn := c.Text("adn") == "--adn"
	if c.Reply.Kind == model.KindUnsupported {
		// just send the user a warning that catching the message may not work
		note := c.note()
		note.Text = c.Locale.Text("commands.kind.warning")
		note.Buttons = []Button{{Label: c.Locale.Text("ui.continue"), Run: func() { c.Do(func() func() { return func() { c.ShowPayload(c.Reply.ChatID, c.Reply.ID, adn) } }) }}}
		return nil
	}
	c.ShowPayload(c.Reply.ChatID, c.Reply.ID, adn)
	return nil
}
