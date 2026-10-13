package wa

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/muleff/WaGo/internal/model"
	whatsmeow "github.com/polymorfa/hypermeow"
)

func box(typ string, body []byte) []byte {
	b := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(b, uint32(8+len(body)))
	copy(b[4:], typ)
	return append(b, body...)
}

func TestMP4Seconds(t *testing.T) {
	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:], 1000)  // time scale
	binary.BigEndian.PutUint32(mvhd[16:], 61500) // duration
	data := append(box("ftyp", []byte("isom0000")), box("moov", append(box("udta", nil), box("mvhd", mvhd)...))...)
	path := filepath.Join(t.TempDir(), "a.mp4")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mp4Seconds(path); got != 62 {
		t.Errorf("mp4Seconds = %d, want 62", got)
	}
}

func TestOggSeconds(t *testing.T) {
	page := func(granule uint64) []byte {
		p := make([]byte, 27)
		copy(p, "OggS")
		binary.LittleEndian.PutUint64(p[6:], granule)
		return p
	}
	data := append(append(page(0), make([]byte, 500)...), page(48000*42+100)...)
	path := filepath.Join(t.TempDir(), "a.ogg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := oggSeconds(path, int64(len(data))); got != 42 {
		t.Errorf("oggSeconds = %d, want 42", got)
	}
}

func TestMP3Seconds(t *testing.T) {
	// A 128 kbit/s frame header after an empty ID3 tag, padded to 160 kB:
	// 10 seconds.
	data := make([]byte, 160000+10)
	copy(data, "ID3\x03\x00\x00\x00\x00\x00\x00")
	copy(data[10:], []byte{0xff, 0xfb, 0x90, 0x00})
	path := filepath.Join(t.TempDir(), "a.mp3")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mp3Seconds(path, int64(len(data))); got != 10 {
		t.Errorf("mp3Seconds = %d, want 10", got)
	}
}

func TestMediaExt(t *testing.T) {
	for _, c := range []struct {
		m    model.Message
		want string
	}{
		{model.Message{Media: model.MediaVoice}, ".ogg"},
		{model.Message{Media: model.MediaDocument, FileName: "Report.PDF"}, ".pdf"},
		{model.Message{Media: model.MediaDocument, FileName: "evil.p/df"}, ".bin"},
		{model.Message{Media: model.MediaAudio, FileType: "audio/mpeg"}, ".mp3"},
		{model.Message{Media: model.MediaVideo}, ".mp4"},
	} {
		if got := mediaExt(&c.m); got != c.want {
			t.Errorf("mediaExt(%+v) = %q, want %q", c.m, got, c.want)
		}
	}
}

func TestVoiceUsesAudioEncryptionAndPTT(t *testing.T) {
	if got := mediaTypes[model.MediaVoice]; got != whatsmeow.MediaAudio {
		t.Fatalf("voice upload type = %q, want WhatsApp audio encryption", got)
	}
	m := &model.Message{
		Media: model.MediaVoice, FileType: "audio/ogg; codecs=opus",
		FileSize: 256, Duration: 7,
	}
	msg, _ := fileMessage(m, upload{}, "", nil, false)
	audio := msg.GetAudioMessage()
	if audio == nil || !audio.GetPTT() || audio.GetSeconds() != 7 ||
		audio.GetMimetype() != "audio/ogg; codecs=opus" {
		t.Fatalf("voice envelope: %+v", audio)
	}
}

func TestUploadKeepsTemporarySourceReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "picked.pdf")
	data := []byte("temporary attachment")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	up := upload{path: path}
	if err := up.holdSource(); err != nil {
		t.Fatal(err)
	}
	defer up.closeSource()
	// Android unlinks the picker copy when SendFile returns; the uploader
	// must keep reading its held descriptor, independent of this pathname.
	_ = os.Remove(path)
	got, err := io.ReadAll(up.source)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("source lost: %q %v", got, err)
	}
	up.closeSource()
	if _, err := up.source.Read(make([]byte, 1)); err == nil {
		t.Fatal("source descriptor leaked")
	}
}

func TestExternalMediaTypeHasNoParameters(t *testing.T) {
	if got := externalMediaType("note.txt"); got != "text/plain" {
		t.Fatalf("intent MIME=%q", got)
	}
}
func TestProviderMIMESurvivesMisleadingExtension(t *testing.T) {
	if got := fileType(model.Attachment{Path: "photo.backup", MIME: "image/jpeg", Media: model.MediaImage}); got != "image/jpeg" {
		t.Fatalf("file MIME=%q", got)
	}
}
