package wa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	waBinary "github.com/polymorfa/hypermeow/binary"
	"github.com/polymorfa/hypermeow/types"
)

const (
	catchNodeMaxPending = 1024
	catchNodeMaxBytes   = 512 << 10
	catchNodeTTL        = 15 * time.Minute
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

// catchSafeNode excludes encrypted content and transport secrets, including
// when they are embedded below a metadata node. It doesn't mutate the stanza
// that hypermeow continues to decode.
func catchSafeNode(node waBinary.Node, depth int) (waBinary.Node, bool) {
	if depth > 8 {
		return waBinary.Node{}, false
	}
	switch strings.ToLower(node.Tag) {
	case "enc", "plaintext", "device-identity", "cipher", "identity",
		"key", "secret", "credentials", "credential", "franking":
		return waBinary.Node{}, false
	}
	clean := waBinary.Node{Tag: node.Tag}
	if len(node.Attrs) > 0 {
		clean.Attrs = make(waBinary.Attrs)
		for key, value := range node.Attrs {
			lower := strings.ToLower(key)
			if lower == "key" || strings.HasSuffix(lower, "_key") || strings.HasPrefix(lower, "key_") ||
				strings.Contains(lower, "secret") || strings.Contains(lower, "token") ||
				strings.Contains(lower, "password") || strings.Contains(lower, "cipher") ||
				strings.Contains(lower, "credential") || strings.Contains(lower, "identity") ||
				strings.Contains(lower, "private") || strings.Contains(lower, "signature") ||
				strings.Contains(lower, "session_key") {
				continue
			}
			clean.Attrs[key] = value
		}
	}
	if children, ok := node.Content.([]waBinary.Node); ok {
		filtered := make([]waBinary.Node, 0, len(children))
		for _, child := range children {
			if safe, ok := catchSafeNode(child, depth+1); ok {
				filtered = append(filtered, safe)
			}
		}
		clean.Content = filtered
	} else if child, ok := node.Content.(waBinary.Node); ok {
		if safe, ok := catchSafeNode(child, depth+1); ok {
			clean.Content = []waBinary.Node{safe}
		}
	} else {
		clean.Content = node.Content
	}
	return clean, true
}

func marshalCatchNodes(nodes []waBinary.Node) ([]byte, error) {
	if nodes == nil {
		return nil, nil
	}
	additional := make([]waBinary.Node, 0, len(nodes))
	for _, child := range nodes {
		if safe, ok := catchSafeNode(child, 0); ok {
			additional = append(additional, safe)
		}
	}
	if len(additional) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(additional)
	if err != nil {
		return nil, err
	}
	if len(data) > catchNodeMaxBytes {
		return nil, fmt.Errorf("additionalNodes exceeds %d bytes", catchNodeMaxBytes)
	}
	return data, nil
}

// onRawCatchNode observes real message stanza siblings via RawNodeHandler.
// The caller retains ownership of node; only sanitized metadata is buffered.
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
	data, err := marshalCatchNodes(children)
	if err != nil || len(data) == 0 {
		return
	}
	from := node.AttrGetter().OptionalJIDOrEmpty("from")
	if recipient := node.AttrGetter().OptionalJIDOrEmpty("recipient"); !recipient.IsEmpty() {
		from = recipient
	}
	chat := ""
	if !from.IsEmpty() {
		chat = b.canonical(b.ctx, from).String()
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
	if rawChat := from.ToNonAD().String(); rawChat != "" && rawChat != chat {
		b.catchNodes.pending[catchNodeKey{chat: rawChat, id: id}] = pendingCatchNode{at: now, data: data}
	}
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
	if !exists && chat != "" {
		if j, err := types.ParseJID(chat); err == nil && !j.IsEmpty() {
			canonKey := catchNodeKey{chat: b.canonical(b.ctx, j).String(), id: id}
			entry, exists = b.catchNodes.pending[canonKey]
			if exists {
				key = canonKey
			}
		}
	}
	if !exists {
		for candidate, possible := range b.catchNodes.pending {
			if candidate.id == id {
				key, entry, exists = candidate, possible, true
				break
			}
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
