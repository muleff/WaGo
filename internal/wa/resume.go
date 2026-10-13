package wa

import (
	"context"
	"github.com/muleff/WaGo/internal/model"
	"time"
)

type resumableSocket interface {
	IsConnected() bool
	ConnectContext(context.Context) error
}

func resumeSocket(ctx context.Context, s resumableSocket) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.IsConnected() {
		return nil
	}
	return s.ConnectContext(ctx)
}

// Resume reconnects the existing device, preserving an in-flight pairing code.
func (b *Backend) Resume() {
	if b.ctx.Err() != nil || !b.resumeInFlight.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer b.resumeInFlight.Store(false)
		cli := b.client()
		if cli == nil || cli.IsConnected() {
			return
		}
		b.pairMu.Lock()
		ctx := b.pairContext
		b.pairMu.Unlock()
		if ctx == nil {
			ctx = b.ctx
		}
		b.emitDisconnectedState()
		if err := retryResume(ctx, cli, time.Second); err != nil && b.ctx.Err() == nil && ctx.Err() == nil {
			b.log.Warnf("foreground reconnect: %v", err)
			b.emitDisconnectedState()
		}
	}()
}
func retryResume(ctx context.Context, s resumableSocket, delay time.Duration) error {
	for {
		err := resumeSocket(ctx, s)
		if s.IsConnected() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = err
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, 15*time.Second)
	}
}

func (b *Backend) emitDisconnectedState() {
	state := model.StateOffline
	if cli := b.client(); cli == nil || cli.Store.ID == nil {
		state = model.StateStarting
	}
	b.emit(model.ConnEvent{State: state})
}

// Mobile uses the client's permanent event handler rather than GetQRChannel,
// whose listener is removed on every socket disconnect.
func (b *Backend) mobileQR(codes []string) {
	b.pairMu.Lock()
	defer b.pairMu.Unlock()
	if b.mobileQRCancel != nil {
		b.mobileQRCancel()
	}
	parent := b.pairContext
	if parent == nil {
		parent = b.ctx
	}
	ctx, cancel := context.WithCancel(parent)
	b.mobileQRCancel = cancel
	if len(codes) == 0 || b.pairCodeRequested.Load() {
		return
	}
	emit := func(code string) { b.emit(model.ConnEvent{State: model.StateQR, QR: code}) }
	emit(codes[0])
	go func() {
		for i := 0; i < len(codes); i++ {
			timeout := 20 * time.Second
			if i == 0 && len(codes) == 6 {
				timeout = time.Minute
			}
			timer := time.NewTimer(timeout)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if b.pairCodeRequested.Load() || b.client().Store.ID != nil {
				return
			}
			if i+1 < len(codes) {
				emit(codes[i+1])
			} else {
				b.emit(model.ConnEvent{State: model.StateQRExpired})
			}
		}
	}()
}
