package androidbridge

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRequestCancellationAndLateResult(t *testing.T) {
	var p requestPool
	ctx, cancel := context.WithCancel(context.Background())
	id, ch := p.begin()
	cancel()
	_, err := p.wait(ctx, id, ch)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if p.complete(id, result{}) {
		t.Fatal("late result accepted")
	}
}
func TestRequestDetachCompletesPending(t *testing.T) {
	var p requestPool
	id, ch := p.begin()
	p.detach()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := p.wait(ctx, id, ch)
	if !errors.Is(err, ErrActivityUnavailable) {
		t.Fatalf("detach: %v", err)
	}
}
func TestRequestCompletesOnce(t *testing.T) {
	var p requestPool
	id, ch := p.begin()
	if !p.complete(id, result{err: ErrPermissionDenied}) {
		t.Fatal("first result rejected")
	}
	if p.complete(id, result{}) {
		t.Fatal("duplicate accepted")
	}
	_, err := p.wait(context.Background(), id, ch)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("denial: %v", err)
	}
}
