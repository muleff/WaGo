package voicerec

import (
	"errors"
	"io"
)

// Stop the producer before finalizing Opus so queued PCM is never truncated.
func finishPCM(stop func(), frames <-chan []byte, encoder io.WriteCloser) (err error) {
	stop()
	defer func() { err = errors.Join(err, encoder.Close()) }()
	for {
		select {
		case buf, ok := <-frames:
			if !ok {
				return nil
			}
			if _, err = encoder.Write(buf); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}
