package wa

import (
	"context"
	"errors"
	"github.com/muleff/WaGo/internal/model"
	"github.com/polymorfa/hypermeow/types/events"
	"testing"
	"time"
)

func TestUnpairedDisconnectDoesNotEnterChatUI(t *testing.T) {
	b, err := Open(t.TempDir(), false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.handle(&events.Disconnected{})
	for _, e := range b.Poll() {
		if state, ok := e.(model.ConnEvent); ok && state.State.LoggedIn() {
			t.Fatalf("unpaired disconnect became logged-in state %v", state.State)
		}
	}
}

type resumeTestSocket struct {
	connected bool
	calls     int
}

func (s *resumeTestSocket) IsConnected() bool { return s.connected }
func (s *resumeTestSocket) ConnectContext(ctx context.Context) error {
	s.calls++
	s.connected = true
	return nil
}
func TestResumeReconnectsSameSocket(t *testing.T) {
	s := &resumeTestSocket{}
	if err := resumeSocket(context.Background(), s); err != nil || s.calls != 1 || !s.connected {
		t.Fatalf("resume: %+v %v", s, err)
	}
	if err := resumeSocket(context.Background(), s); err != nil || s.calls != 1 {
		t.Fatalf("connected socket restarted: %+v %v", s, err)
	}
}

func TestMobilePairingEventsRemainAfterDisconnect(t *testing.T) {
	b, err := Open(t.TempDir(), false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.handle(&events.Disconnected{})
	b.Poll()
	b.handle(&events.QR{Codes: []string{"fresh QR"}})
	found := false
	for _, e := range b.Poll() {
		if state, ok := e.(model.ConnEvent); ok && state.State == model.StateQR && state.QR == "fresh QR" {
			found = true
		}
	}
	if !found {
		t.Fatal("pairing events stopped after disconnect")
	}
}

type flakyResumeSocket struct {
	attempts  int
	connected bool
}

func (s *flakyResumeSocket) IsConnected() bool { return s.connected }
func (s *flakyResumeSocket) ConnectContext(ctx context.Context) error {
	s.attempts++
	if s.attempts < 3 {
		return errors.New("offline")
	}
	s.connected = true
	return nil
}
func TestResumeRetriesUntilConnectivityReturns(t *testing.T) {
	s := new(flakyResumeSocket)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := retryResume(ctx, s, time.Millisecond); err != nil || s.attempts != 3 {
		t.Fatalf("reconnect attempts=%d err=%v", s.attempts, err)
	}
}
