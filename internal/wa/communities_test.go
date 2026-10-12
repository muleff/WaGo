package wa

import (
	"testing"

	"github.com/polymorfa/hypermeow/store"
	"github.com/polymorfa/hypermeow/types"
	"github.com/polymorfa/hypermeow/types/events"
)

// TestJoinCommunity checks that joining a community stores its parent group
// and announcements in their places, so neither lists as a plain group, and
// that groups linked to it later (or unlinked) move in and out.
func TestJoinCommunity(t *testing.T) {
	b := testBackend(t)
	comm := types.NewJID("c", types.GroupServer)
	ann := types.NewJID("ann", types.GroupServer)
	join := func(g types.GroupInfo) { b.handle(&events.JoinedGroup{GroupInfo: g}) }
	join(types.GroupInfo{JID: comm, GroupName: types.GroupName{Name: "Community"}, GroupParent: types.GroupParent{IsParent: true}})
	join(types.GroupInfo{JID: ann, GroupName: types.GroupName{Name: "Community"},
		GroupLinkedParent: types.GroupLinkedParent{LinkedParentJID: comm}, GroupIsDefaultSub: types.GroupIsDefaultSub{IsDefaultSubGroup: true}})

	for _, c := range b.Chats() {
		if c.ID == comm.String() {
			t.Error("the community's parent group lists as a chat")
		}
	}
	cs := b.Communities()
	if len(cs) != 1 || cs[0].Announcements != ann.String() || len(cs[0].Groups) != 0 {
		t.Fatalf("communities = %+v, want one with announcements %s and no groups", cs, ann)
	}

	sub := types.NewJID("sub", types.GroupServer)
	join(types.GroupInfo{JID: sub, GroupName: types.GroupName{Name: "Sub"}})
	link := &types.GroupLinkChange{Type: types.GroupLinkChangeTypeSub, Group: types.GroupLinkTarget{JID: sub}}
	b.handle(&events.GroupInfo{JID: comm, Link: link})
	if cs := b.Communities(); len(cs) != 1 || len(cs[0].Groups) != 1 || cs[0].Groups[0] != sub.String() {
		t.Fatalf("after linking, communities = %+v, want %s in it", cs, sub)
	}
	b.handle(&events.GroupInfo{JID: comm, Unlink: link})
	if cs := b.Communities(); len(cs) != 1 || len(cs[0].Groups) != 0 {
		t.Fatalf("after unlinking, communities = %+v, want no groups", cs)
	}
}

// TestExitCommunitySubgroup keeps the chat until the user deletes it, but
// removes it from the community immediately on a confirmed exit.
func TestExitCommunitySubgroup(t *testing.T) {
	b := testBackend(t)
	comm := types.NewJID("parent", types.GroupServer)
	sub := types.NewJID("topic", types.GroupServer)
	b.handle(&events.JoinedGroup{GroupInfo: types.GroupInfo{
		JID: comm, GroupName: types.GroupName{Name: "Parent"}, GroupParent: types.GroupParent{IsParent: true},
	}})
	b.handle(&events.JoinedGroup{GroupInfo: types.GroupInfo{
		JID: sub, GroupName: types.GroupName{Name: "Topic"},
		GroupLinkedParent: types.GroupLinkedParent{LinkedParentJID: comm},
	}})
	if got := b.Communities(); len(got) != 1 || len(got[0].Groups) != 1 {
		t.Fatalf("joined community = %+v", got)
	}
	b.markGroupExited(b.ctx, sub.String())
	if c := b.chat(sub.String()); c == nil || !c.Exited {
		t.Fatalf("exited chat = %+v, want Exited", c)
	}
	if got := b.Communities(); len(got) != 0 {
		t.Fatalf("after exit community = %+v, want none", got)
	}
	if c := b.chat(sub.String()); c == nil {
		t.Fatal("leaving should keep chat history until deletion")
	}
	b.DeleteChat(sub.String())
	if c := b.chat(sub.String()); c != nil {
		t.Fatalf("deleted chat still stored: %+v", c)
	}
	if got := b.Communities(); len(got) != 0 {
		t.Fatalf("deleted chat reappeared in community: %+v", got)
	}
}

// TestJoinedGroupsAfterOfflineLeave checks that leaving on another device
// changes cached membership without discarding conversations or children
// still joined in the same community.
func TestJoinedGroupsAfterOfflineLeave(t *testing.T) {
	b := testBackend(t)
	parent := types.NewJID("p", types.GroupServer)
	left := types.NewJID("left", types.GroupServer)
	active := types.NewJID("active", types.GroupServer)
	for _, g := range []types.GroupInfo{
		{JID: parent, GroupName: types.GroupName{Name: "P"}, GroupParent: types.GroupParent{IsParent: true}},
		{JID: left, GroupName: types.GroupName{Name: "L"}, GroupLinkedParent: types.GroupLinkedParent{LinkedParentJID: parent}},
		{JID: active, GroupName: types.GroupName{Name: "A"}, GroupLinkedParent: types.GroupLinkedParent{LinkedParentJID: parent}},
	} {
		b.handle(&events.JoinedGroup{GroupInfo: g})
	}
	if err := b.store.syncJoinedGroups(b.ctx, []string{parent.String(), active.String()}); err != nil {
		t.Fatal(err)
	}
	if c := b.chat(left.String()); c == nil || !c.Exited {
		t.Fatalf("offline-exited chat = %+v, want Exited", c)
	}
	if c := b.chat(active.String()); c == nil || c.Exited {
		t.Fatalf("still-joined chat = %+v, want not Exited", c)
	}
	if got := b.Communities(); len(got) != 1 || len(got[0].Groups) != 1 || got[0].Groups[0] != active.String() {
		t.Fatalf("remaining community = %+v", got)
	}
	if err := b.store.syncJoinedGroups(b.ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.Communities(); len(got) != 0 {
		t.Fatalf("left all groups but communities = %+v", got)
	}
	if err := b.store.syncJoinedGroups(b.ctx, []string{parent.String(), left.String()}); err != nil {
		t.Fatal(err)
	}
	if c := b.chat(left.String()); c == nil || c.Exited {
		t.Fatalf("rejoined chat = %+v, want not Exited", c)
	}
}

func TestOwnLeaveEventMarksExited(t *testing.T) {
	b := testBackend(t)
	me := types.NewJID("123", types.DefaultUserServer)
	b.useDevice(&store.Device{ID: &me})
	group := types.NewJID("team", types.GroupServer)
	b.handle(&events.JoinedGroup{GroupInfo: types.GroupInfo{
		JID: group, GroupName: types.GroupName{Name: "Team"},
		Participants: []types.GroupParticipant{{JID: me}},
	}})
	if c := b.chat(group.String()); c == nil || c.Exited {
		t.Fatalf("newly joined group = %+v", c)
	}
	b.handle(&events.GroupInfo{JID: group, Leave: []types.JID{me}})
	if c := b.chat(group.String()); c == nil || !c.Exited {
		t.Fatalf("group after own leave event = %+v", c)
	}
	if got := b.store.members(b.ctx, group.String()); len(got) != 0 {
		t.Fatalf("participants after leaving = %+v, want none", got)
	}
	// A later explicit join resets the exit marker.
	b.handle(&events.GroupInfo{JID: group, Join: []types.JID{me}})
	if c := b.chat(group.String()); c == nil || c.Exited {
		t.Fatalf("group after rejoin = %+v", c)
	}
}

// TestRename checks that a chat's stored name is only reported changed when
// it is.
func TestRename(t *testing.T) {
	b := testBackend(t)
	jid := types.NewJID("111", types.HiddenUserServer).String()
	if err := b.store.ensureChat(b.ctx, b.db, jid, false, "~Push"); err != nil {
		t.Fatal(err)
	}
	if !b.store.rename(b.ctx, jid, "Saved") {
		t.Error("renaming to a new name reports no change")
	}
	if b.store.rename(b.ctx, jid, "Saved") {
		t.Error("renaming to the same name reports a change")
	}
	if b.store.rename(b.ctx, types.NewJID("222", types.HiddenUserServer).String(), "X") {
		t.Error("renaming a chat that isn't stored reports a change")
	}
	if c := b.chat(jid); c == nil || c.Name != "Saved" {
		t.Errorf("stored chat = %+v, want named Saved", c)
	}
}
