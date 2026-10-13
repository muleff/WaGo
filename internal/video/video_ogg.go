package video

import (
	"io"
	"os"

	"github.com/tphakala/go-opus/oggopus"
)

// scanVoice decodes the whole Ogg/Opus stream once to learn its length in
// samples per channel. Shared with the Android player; pure Go.
func scanVoice(f *os.File) (total int64, ch int, err error) {
	dec, err := oggopus.NewDecoder(f)
	if err != nil {
		return 0, 0, err
	}
	ch = dec.Info().Channels
	n, err := io.Copy(io.Discard, dec)
	if err != nil {
		return 0, 0, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}
	return n / int64(2*ch), ch, nil
}
