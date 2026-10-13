package ui

import (
	"image"
	"testing"
	"time"
)

func TestImageCachePublishesImmutableFrameEntry(t *testing.T) {
	c := newImageCache(10, 1024)
	finish := make(chan struct{})
	published := make(chan struct{}, 1)
	c.invalidate = func() { published <- struct{}{} }
	load := func() []byte { return []byte{1} }
	decode := func([]byte) (image.Image, bool) {
		<-finish
		return image.NewRGBA(image.Rect(0, 0, 2, 3)), false
	}
	frameEntry := c.getWith("photo", load, decode)
	close(finish)
	select {
	case <-published:
	case <-time.After(2 * time.Second):
		t.Fatal("image was not published")
	}
	// A frame can still be reading this entry when the decoder finishes.
	// Publishing must not mutate any of its rendering fields underneath it.
	if frameEntry.state != imgLoading {
		t.Fatal("decoder mutated an entry already handed to the frame")
	}
	ready := c.getWith("photo", load, decode)
	if ready.state != imgReady || ready.size != image.Pt(2, 3) || ready.bytes != 24 || c.bytes != 24 {
		t.Fatalf("next frame did not receive the decoded image: %+v, bytes %d", ready, c.bytes)
	}
	c.forget("photo")
	if c.bytes != 0 {
		t.Fatalf("decoded bytes not released from cache accounting: %d", c.bytes)
	}
}

func TestImageCacheDoesNotResurrectForgottenLoad(t *testing.T) {
	c := newImageCache(10, 1024)
	finish := make(chan struct{})
	published := make(chan struct{}, 1)
	c.invalidate = func() { published <- struct{}{} }
	c.getWith("old", func() []byte { return []byte{1} }, func([]byte) (image.Image, bool) {
		<-finish
		return image.NewRGBA(image.Rect(0, 0, 2, 3)), false
	})
	c.forget("old")
	close(finish)
	select {
	case <-published:
	case <-time.After(2 * time.Second):
		t.Fatal("image worker did not finish")
	}
	if _, ok := c.m["old"]; ok || c.bytes != 0 {
		t.Fatal("a forgotten load returned to the image cache")
	}
}
