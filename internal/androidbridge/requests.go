// Package androidbridge connects Go workers to Android system UI.
package androidbridge

import (
	"context"
	"errors"
	"sync"
)

var ErrActivityUnavailable = errors.New("Android activity is unavailable")
var ErrPermissionDenied = errors.New("microphone permission was denied")

type PickedFile struct{ Path, Name, MIME string }
type result struct {
	files []PickedFile
	err   error
}
type requestPool struct {
	mu      sync.Mutex
	next    uint64
	pending map[uint64]chan result
}

func (p *requestPool) begin() (uint64, chan result) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pending == nil {
		p.pending = make(map[uint64]chan result)
	}
	p.next++
	ch := make(chan result, 1)
	p.pending[p.next] = ch
	return p.next, ch
}
func (p *requestPool) complete(id uint64, r result) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch, ok := p.pending[id]
	if !ok {
		return false
	}
	delete(p.pending, id)
	ch <- r
	return true
}
func (p *requestPool) detach() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, ch := range p.pending {
		delete(p.pending, id)
		ch <- result{err: ErrActivityUnavailable}
	}
}
func (p *requestPool) wait(ctx context.Context, id uint64, ch chan result) ([]PickedFile, error) {
	select {
	case r := <-ch:
		return r.files, r.err
	case <-ctx.Done():
		p.complete(id, result{err: ctx.Err()})
		return nil, ctx.Err()
	}
}
