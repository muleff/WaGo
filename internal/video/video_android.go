//go:build android

package video

import (
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/tphakala/go-opus/oggopus"
)

// This backend plays sound files the pure-Go oggopus decoder understands
// (voice notes) through malgo, with no external player. Video pictures
// still need a system player (see Open), so Open stays unsupported.
func start(p *Player, path string) error {
	if !p.audio {
		return ErrUnsupported
	}
	return startAudio(p, path)
}

// androidAudio streams one Ogg/Opus file. A single goroutine owns the
// decoder and the device; the malgo callback only pulls queued PCM.
type androidAudio struct {
	p      *Player
	file   *os.File
	ch     int
	total  int64 // samples per channel
	dec    *oggopus.Decoder
	mctx   *malgo.AllocatedContext
	device *malgo.Device
	queue  chan []byte
	played atomic.Int64
	eof    atomic.Bool
	muted  atomic.Bool
	ended  atomic.Bool
	paused atomic.Bool
	// dropCarry asks the audio thread to forget its partial chunk
	// (seek path); carry itself is audio-thread-local.
	dropCarry atomic.Bool
	carry     []byte // partial chunk, audio thread only
}

func startAudio(p *Player, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	total, ch, err := scanVoice(f)
	if err != nil {
		f.Close()
		return err
	}
	a := &androidAudio{
		p:     p,
		file:  f,
		ch:    ch,
		total: total,
		queue: make(chan []byte, 32),
	}
	a.paused.Store(true)
	if err := a.rewind(0); err != nil {
		f.Close()
		return err
	}
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		f.Close()
		return err
	}
	a.mctx = mctx
	config := malgo.DefaultDeviceConfig(malgo.Playback)
	config.SampleRate = 48000
	config.Playback.Format = malgo.FormatS16
	config.Playback.Channels = uint32(ch)
	device, err := malgo.InitDevice(mctx.Context, config, malgo.DeviceCallbacks{
		Data: a.fill,
	})
	if err != nil {
		mctx.Uninit()
		mctx.Free()
		f.Close()
		return err
	}
	a.device = device
	p.setStatus(Status{Ready: true, Paused: true, Dur: samplesToDuration(total)})
	go a.loop()
	return nil
}

// rewind reopens the decoder at the given sample.
func (a *androidAudio) rewind(sample int64) error {
	if _, err := a.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	dec, err := oggopus.NewDecoder(a.file)
	if err != nil {
		return err
	}
	a.dec = dec
	skip := sample * int64(2*a.ch)
	buf := make([]byte, 32*1024)
	for skip > 0 {
		n := min(int64(len(buf)), skip)
		if _, err := io.ReadFull(dec, buf[:n]); err != nil {
			return err
		}
		skip -= n
	}
	return nil
}

func samplesToDuration(samples int64) time.Duration {
	return time.Duration(samples) * time.Second / 48000
}

func (a *androidAudio) status() Status {
	st := Status{
		Ready:  true,
		Paused: a.paused.Load(),
		Ended:  a.ended.Load(),
		Pos:    samplesToDuration(a.played.Load()),
		Dur:    samplesToDuration(a.total),
	}
	return st
}

// fill runs on the audio thread: never block, never allocate much.
func (a *androidAudio) fill(out, _ []byte, _ uint32) {
	if a.dropCarry.CompareAndSwap(true, false) {
		a.carry = nil
	}
	if a.muted.Load() {
		// Keep consuming so the position stays in sync while silent.
		a.discard(len(out))
		clear(out)
		a.played.Add(int64(len(out)) / int64(2*a.ch))
		return
	}
	off := 0
	for off < len(out) {
		if len(a.carry) == 0 {
			select {
			case chunk, ok := <-a.queue:
				if !ok || len(chunk) == 0 {
					clear(out[off:])
					a.played.Add(int64(len(out)) / int64(2*a.ch))
					return
				}
				a.carry = chunk
			default:
				clear(out[off:])
				a.played.Add(int64(len(out)) / int64(2*a.ch))
				return
			}
		}
		n := copy(out[off:], a.carry)
		a.carry = a.carry[n:]
		off += n
	}
	a.played.Add(int64(len(out)) / int64(2*a.ch))
}

func (a *androidAudio) loop() {
	defer a.device.Uninit()
	defer a.mctx.Uninit()
	defer a.mctx.Free()
	defer a.file.Close()
	chunk := make([]byte, 3840) // ~20 ms of 48 kHz mono s16
	for {
		select {
		case <-a.p.done:
			return
		case c := <-a.p.cmds:
			a.command(c)
		default:
			if a.paused.Load() || a.eof.Load() {
				a.publish()
				select {
				case <-a.p.done:
					return
				case c := <-a.p.cmds:
					a.command(c)
				}
				continue
			}
			n, err := a.dec.Read(chunk)
			if n > 0 {
				buf := make([]byte, n)
				copy(buf, chunk[:n])
				select {
				case a.queue <- buf:
				case <-a.p.done:
					return
				}
			}
			if err != nil {
				a.eof.Store(true)
				a.publish()
			}
		}
	}
}

func (a *androidAudio) command(c command) {
	switch c.kind {
	case cmdPlay:
		if a.ended.Load() {
			if err := a.rewind(0); err != nil {
				a.p.setStatus(Status{Ready: true, Paused: true, Dur: samplesToDuration(a.total), Err: err})
				return
			}
			a.drain()
			a.played.Store(0)
			a.eof.Store(false)
			a.ended.Store(false)
		}
		if a.paused.Load() {
			if err := a.device.Start(); err != nil {
				a.p.setStatus(Status{Ready: true, Paused: true, Dur: samplesToDuration(a.total), Err: err})
				return
			}
			a.paused.Store(false)
		}
	case cmdPause:
		if !a.paused.Load() {
			a.device.Stop()
			a.paused.Store(true)
		}
	case cmdSeek:
		sample := int64(float64(c.pos) / float64(time.Second) * 48000)
		sample = min(max(sample, 0), a.total)
		if err := a.rewind(sample); err != nil {
			a.p.setStatus(Status{Ready: true, Paused: true, Dur: samplesToDuration(a.total), Err: err})
			return
		}
		a.drain()
		a.played.Store(sample)
		a.eof.Store(false)
		a.ended.Store(false)
	case cmdMute:
		a.muted.Store(c.on)
	}
	a.publish()
}

// drain drops queued audio, e.g. after a seek.
func (a *androidAudio) drain() {
	a.dropCarry.Store(true)
	for {
		select {
		case <-a.queue:
		default:
			return
		}
	}
}

// discard drops up to n bytes of queued audio on the mute path, so the
// position stays in sync while silent. Audio thread only, like carry.
func (a *androidAudio) discard(n int) {
	for n > 0 {
		if len(a.carry) > 0 {
			m := min(len(a.carry), n)
			a.carry = a.carry[m:]
			n -= m
			continue
		}
		select {
		case a.carry = <-a.queue:
		default:
			return
		}
	}
}

// publish refreshes the status, marking the end once every sample played.
func (a *androidAudio) publish() {
	if a.eof.Load() && a.played.Load() >= a.total && !a.ended.Load() {
		a.ended.Store(true)
		a.device.Stop()
		a.paused.Store(true)
	}
	a.p.setStatus(a.status())
}
