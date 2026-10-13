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
		Pos:    samplesToDuration(min(a.played.Load(), a.total)),
		Dur:    samplesToDuration(a.total),
	}
	return st
}

// fill runs on the audio thread: never block, never allocate much.
func (a *androidAudio) fill(out, _ []byte, _ uint32) {
	if a.dropCarry.CompareAndSwap(true, false) {
		a.carry = nil
	}
	n := takePCM(out, &a.carry, a.queue)
	if a.muted.Load() {
		clear(out)
	}
	a.played.Add(int64(n) / int64(2*a.ch))
}

func (a *androidAudio) loop() {
	defer func() { a.device.Stop(); a.device.Uninit(); a.mctx.Uninit(); a.mctx.Free(); a.file.Close() }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	chunk := make([]byte, 3840)
	var pending []byte
	for {
		if !a.paused.Load() && !a.eof.Load() && pending == nil {
			n, err := a.dec.Read(chunk)
			if n > 0 {
				pending = append([]byte(nil), chunk[:n]...)
			}
			if err != nil {
				a.eof.Store(true)
				if err != io.EOF {
					a.p.setStatus(Status{Ready: true, Paused: true, Err: err})
					return
				}
			}
		}
		var output chan []byte
		if !a.paused.Load() && pending != nil {
			output = a.queue
		}
		select {
		case <-a.p.done:
			return
		case c := <-a.p.cmds:
			if c.kind == cmdSeek || c.kind == cmdPlay && a.ended.Load() {
				pending = nil
			}
			a.command(c)
		case output <- pending:
			pending = nil
		case <-ticker.C:
			a.publish()
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
		wasPlaying := !a.paused.Load()
		if wasPlaying {
			a.device.Stop()
		}
		defer func() {
			if wasPlaying {
				if err := a.device.Start(); err != nil {
					a.paused.Store(true)
					a.p.setStatus(Status{Ready: true, Paused: true, Err: err})
				}
			}
		}()
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
