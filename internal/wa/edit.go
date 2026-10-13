package wa

import (
	"errors"
	"fmt"
	"strings"
	"time"

	whatsmeow "github.com/polymorfa/hypermeow"
	"github.com/polymorfa/hypermeow/proto/waCommon"
	"github.com/polymorfa/hypermeow/proto/waE2E"
	"github.com/polymorfa/hypermeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/muleff/WaGo/internal/model"
)

// EditText implements model.Backend. The stored text refers to people as
// "@<user>"; the composer shows "@Name".
func (b *Backend) EditText(m *model.Message) (string, []model.MentionRef) {
	ctx := b.ctx
	r, ok := b.store.message(ctx, m.ChatID, m.ID)
	if !ok {
		return stripMarks(m.Text), nil
	}
	text := r.Text
	var refs []model.MentionRef
	for _, s := range strings.Split(r.mentions, ",") {
		switch {
		case s == "":
		case s == mentionAll:
			refs = append(refs, model.MentionRef{Name: "all", ID: "@all"})
		case strings.HasPrefix(s, groupMentionPrefix):
			jid, subject, _ := strings.Cut(s[len(groupMentionPrefix):], ":")
			if jid == m.ChatID {
				text = strings.ReplaceAll(text, "@"+jid, "@admin")
				refs = append(refs, model.MentionRef{Name: "admin", ID: "@admin"})
			} else {
				text = strings.ReplaceAll(text, "@"+jid, "@"+subject)
			}
		default:
			j, err := types.ParseJID(s)
			if err != nil || j.User == "" || !strings.Contains(text, "@"+j.User) {
				continue
			}
			name := b.senderName(ctx, j, "")
			text = strings.ReplaceAll(text, "@"+j.User, "@"+name)
			refs = append(refs, model.MentionRef{Name: name, ID: j.ToNonAD().String()})
		}
	}
	return text, refs
}

// stripMarks removes the isolate marks around resolved mentions.
func stripMarks(s string) string {
	return strings.NewReplacer("⁨", "", "⁩", "", string(model.MentionNotifies), "",
		string(model.MentionAdmins), "").Replace(s)
}

// ReplicateMessage implements model.Backend. It reproduces reply's text in a
// group using the group-status edit flow.
func (b *Backend) ReplicateMessage(chatID string, d model.Draft, reply *model.Message) error {
	if reply == nil {
		return errors.New("reply to a message first")
	}
	if reply.ChatID != chatID {
		return errors.New("the replied-to message is in a different chat")
	}
	cli := b.connected()
	jid, err := types.ParseJID(chatID)
	if err != nil || jid.Server != types.GroupServer {
		return errors.New("message replication only works in groups")
	}
	if cli == nil {
		return errors.New("not connected")
	}
	ctx := b.ctx
	// store reply id
	id := reply.ID
	// send a groupStatusMessagev2
	groupStatus := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: proto.String(""),
					ContextInfo: &waE2E.ContextInfo{
						IsGroupStatus: proto.Bool(true),
					},
				},
			},
		},
	}
	resp, err := cli.SendMessage(ctx, jid, groupStatus)
	if err != nil {
		return fmt.Errorf("send replication message: %w", err)
	}
	tempID := resp.ID

	protocolMessage := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ProtocolMessage: &waE2E.ProtocolMessage{
					Key: &waCommon.MessageKey{
						ID:        proto.String(tempID),
						RemoteJID: proto.String(chatID),
						FromMe:    proto.Bool(true),
					},
					Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
					EditedMessage: &waE2E.Message{
						ExtendedTextMessage: &waE2E.ExtendedTextMessage{
							Text: proto.String(d.Text),
							ContextInfo: &waE2E.ContextInfo{
								IsGroupStatus: proto.Bool(false),
							},
						},
					},
				},
			},
		},
	}
	resp2, err := cli.SendMessage(ctx, jid, protocolMessage, whatsmeow.SendRequestExtra{
		ID: types.MessageID(id),
	})
	if err != nil {
		_, revokeErr := cli.SendMessage(ctx, jid, cli.BuildRevoke(jid, types.EmptyJID, tempID))
		if revokeErr != nil {
			return errors.Join(fmt.Errorf("edit replication message: %w", err), fmt.Errorf("revoke temporary message: %w", revokeErr))
		}
		return fmt.Errorf("edit replication message: %w", err)
	}
	tempID2 := resp2.ID
	time.Sleep(100 * time.Millisecond) // 100 ms
	var revokeErr error
	if _, err := cli.SendMessage(ctx, jid, cli.BuildRevoke(jid, types.EmptyJID, tempID)); err != nil {
		revokeErr = errors.Join(revokeErr, fmt.Errorf("revoke temporary message: %w", err))
	}
	if _, err := cli.SendMessage(ctx, jid, cli.BuildRevoke(jid, types.EmptyJID, tempID2)); err != nil {
		revokeErr = errors.Join(revokeErr, fmt.Errorf("revoke edit message: %w", err))
	}
	return revokeErr
}

// Edit implements model.Backend.
func (b *Backend) Edit(m *model.Message, d model.Draft) {
	cli := b.connected()
	jid, err := types.ParseJID(m.ChatID)
	if cli == nil || err != nil {
		b.emit(model.NoticeEvent{Text: "Couldn't edit the message."})
		return
	}
	ctx := b.ctx
	d.Reply = nil
	ci := b.draftContext(m.ChatID, d, &model.Message{ChatID: m.ChatID})
	var content *waE2E.Message
	switch m.Media {
	case model.MediaNone:
		content = &waE2E.Message{Conversation: proto.String(d.Text)}
		if ci != nil {
			content = &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String(d.Text), ContextInfo: ci}}
		}
	case model.MediaImage, model.MediaVideo, model.MediaGIF:
		// A caption goes with the media message it belongs to.
		media, blob, err := b.store.mediaBlob(ctx, m.ChatID, m.ID)
		if err != nil || len(blob) == 0 {
			b.emit(model.NoticeEvent{Text: "Couldn't edit the caption."})
			return
		}
		if media == model.MediaImage {
			var im waE2E.ImageMessage
			if proto.Unmarshal(blob, &im) != nil {
				return
			}
			im.Caption, im.ContextInfo = proto.String(d.Text), ci
			content = &waE2E.Message{ImageMessage: &im}
		} else {
			var vm waE2E.VideoMessage
			if proto.Unmarshal(blob, &vm) != nil {
				return
			}
			vm.Caption, vm.ContextInfo = proto.String(d.Text), ci
			content = &waE2E.Message{VideoMessage: &vm}
		}
	default:
		return
	}
	now := time.Now()
	if err := b.store.editText(ctx, m.ChatID, m.ID, d.Text, now, marshal(content)); err != nil {
		b.log.Errorf("store edit of %s: %v", m.ID, err)
	}
	b.emitMessage(m.ChatID, m.ID)
	b.emitChat(m.ChatID)
	msg := cli.BuildEdit(jid, m.ID, content)
	go func() {
		if _, err := cli.SendMessage(ctx, jid, msg); err != nil {
			b.log.Warnf("edit %s in %s: %v", m.ID, m.ChatID, err)
			b.emit(model.NoticeEvent{Text: "Couldn't edit the message."})
		}
	}()
}

// Versions implements model.Backend.
func (b *Backend) Versions(m *model.Message) []model.Version {
	ctx := b.ctx
	texts, mentions, times, err := b.store.versions(ctx, m.ChatID, m.ID)
	if err != nil {
		b.log.Warnf("versions of %s: %v", m.ID, err)
		return nil
	}
	out := make([]model.Version, len(texts))
	for i, t := range texts {
		if mentions[i] != "" {
			t = b.replaceMentions(ctx, m.ChatID, t, mentions[i])
		} else if strings.Contains(t, "@") {
			t = b.guessMentions(ctx, t)
		}
		out[i] = model.Version{Text: t, Time: times[i]}
	}
	return out
}
