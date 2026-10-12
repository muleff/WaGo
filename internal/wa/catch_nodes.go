package wa

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	waBinary "github.com/polymorfa/hypermeow/binary"
	"github.com/polymorfa/hypermeow/types/events"
	"github.com/skidy89/openWA/internal/model"
)

const (
	catchNodeMaxPending = 64
	catchNodeMaxBytes   = 128 << 10
	catchNodeTTL        = 2 * time.Minute
)

type catchNodeKey struct{ chat, id string }

type pendingCatchNode struct {
	at   time.Time
	data []byte
}

// catchNodeBuffer correlates a raw stanza with its subsequently decrypted event.
// Encrypted message bytes never enter this buffer.
type catchNodeBuffer struct {
	mu      sync.Mutex
	pending map[catchNodeKey]pendingCatchNode
}

// onRawCatchNode observes real message stanza siblings via RawNodeHandler.
// The caller retains ownership of node; only its JSON representation is saved.
func (b *Backend) onRawCatchNode(_ context.Context, node *waBinary.Node) {
	if node == nil || node.Tag != "message" {
		return
	}
	id := node.AttrGetter().OptionalString("id")
	if id == "" {
		return
	}
	children := node.GetChildren()
	if children == nil {
		return
	}
	additional := make([]waBinary.Node, 0, len(children))
	for _, child := range children {
		switch child.Tag {
		case "enc", "plaintext", "participants", "device-identity", "unavailable", "to", "payload":
			// Message ciphertext, recipient data and transport framing are
			// unrelated to the additionalNodes sent with a message.
			continue
		}
		additional = append(additional, child)
	}
	data, err := json.Marshal(additional)
	if err != nil || len(data) > catchNodeMaxBytes {
		return
	}
	from := node.AttrGetter().OptionalJIDOrEmpty("from")
	chat := ""
	if !from.IsEmpty() {
		chat = from.ToNonAD().String()
	}
	key := catchNodeKey{chat: chat, id: id}
	now := b.now()
	b.catchNodes.mu.Lock()
	defer b.catchNodes.mu.Unlock()
	if b.catchNodes.pending == nil {
		b.catchNodes.pending = make(map[catchNodeKey]pendingCatchNode)
	}
	for oldKey, pending := range b.catchNodes.pending {
		if now.Sub(pending.at) > catchNodeTTL {
			delete(b.catchNodes.pending, oldKey)
		}
	}
	if len(b.catchNodes.pending) >= catchNodeMaxPending {
		var oldest catchNodeKey
		var oldAt time.Time
		for candidate, pending := range b.catchNodes.pending {
			if oldAt.IsZero() || pending.at.Before(oldAt) {
				oldest, oldAt = candidate, pending.at
			}
		}
		delete(b.catchNodes.pending, oldest)
	}
	b.catchNodes.pending[key] = pendingCatchNode{at: now, data: data}
}

// popCatchNodes only consumes stanza metadata after a message has been stored.
// Direct chat identifiers may differ between the stanza and the model's
// canonical LID; in that case an ID-only match is accepted if unambiguous.
func (b *Backend) popCatchNodes(chat, id string) []byte {
	b.catchNodes.mu.Lock()
	defer b.catchNodes.mu.Unlock()
	if b.catchNodes.pending == nil {
		return nil
	}
	key := catchNodeKey{chat: chat, id: id}
	entry, exists := b.catchNodes.pending[key]
	if !exists {
		for candidate, possible := range b.catchNodes.pending {
			if candidate.id != id {
				continue
			}
			if exists {
				return nil // two chats used the same ID: don't guess
			}
			key, entry, exists = candidate, possible, true
		}
	}
	if !exists {
		return nil
	}
	delete(b.catchNodes.pending, key)
	if b.now().Sub(entry.at) > catchNodeTTL {
		return nil
	}
	return entry.data
}

func (b *Backend) saveCatchNodes(evt *events.Message) {
	chat := b.canonical(b.ctx, evt.Info.Chat).String()
	data := b.popCatchNodes(chat, evt.Info.ID)
	if len(data) == 0 {
		return
	}
	if _, err := b.db.ExecContext(b.ctx, `UPDATE wz_messages
		SET additional_nodes = COALESCE(additional_nodes, ?)
		WHERE chat = ? AND id = ? AND kind != ?`, data, chat, evt.Info.ID, int(model.KindDeleted)); err != nil {
		b.log.Warnf("store additionalNodes for %s in %s: %v", evt.Info.ID, chat, err)
	}
}
