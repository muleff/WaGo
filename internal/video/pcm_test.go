package video

import (
	"bytes"
	"testing"
)

func TestPCMUnderflowDoesNotAdvancePosition(t *testing.T) {
	q := make(chan []byte, 2)
	q <- []byte{1, 2}
	out := make([]byte, 8)
	var carry []byte
	n := takePCM(out, &carry, q)
	if n != 2 || !bytes.Equal(out, []byte{1, 2, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("consumed %d bytes: %v", n, out)
	}
	if n = takePCM(out, &carry, q); n != 0 {
		t.Fatalf("underflow consumed %d bytes", n)
	}
}
func TestPCMRetainsPartialChunk(t *testing.T) {
	q := make(chan []byte, 1)
	q <- []byte{1, 2, 3, 4}
	var carry []byte
	out := make([]byte, 2)
	if takePCM(out, &carry, q) != 2 || !bytes.Equal(out, []byte{1, 2}) {
		t.Fatal("first chunk")
	}
	if takePCM(out, &carry, q) != 2 || !bytes.Equal(out, []byte{3, 4}) {
		t.Fatal("remaining chunk")
	}
}

func TestAudioStartsWithPlayCommand(t *testing.T) {
	p := newPlayerState(nil, true)
	select {
	case c := <-p.cmds:
		if c.kind != cmdPlay {
			t.Fatalf("first command=%v", c.kind)
		}
	default:
		t.Fatal("first tap does not start playback")
	}
}
