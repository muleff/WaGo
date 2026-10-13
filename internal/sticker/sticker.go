// Package sticker makes WhatsApp stickers: a picture fitted into a 512x512
// transparent canvas, with meme text if you like, as a lossless WebP. It
// has its own WebP encoder (vp8l.go), since golang.org/x/image only
// decodes WebP, and libwebp needs cgo or, translated to Go
// (github.com/gen2brain/webp), adds megabytes to the executable and
// registers a second WebP decoder for image.Decode.
package sticker

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	"image/draw"

	_ "golang.org/x/image/webp"
	_ "image/gif" // the formats a sticker can be made from
	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"

	"github.com/muleff/WaGo/internal/photo"
)

type stickerMetadata struct {
	PackID          string   `json:"sticker-pack-id"`
	PackName        string   `json:"sticker-pack-name"`
	Publisher       string   `json:"sticker-pack-publisher"`
	AndroidLink     *string  `json:"android-app-store-link"`
	IOSLink         *string  `json:"ios-app-store-link"`
	AiSticker       *int     `json:"is-ai-sticker"`
	Emojis          []string `json:"emojis"`
	UserCreatedPack *int     `json:"is-from-user-created-pack"`
	AvatarSticker   *int     `json:"is-avatar-sticker"`
	Premium         *int     `json:"premium"`
}

type Pack struct {
	Name       string
	Author     string
	AI_STICKER bool
	PREMIUM    bool
}

type webpChunk struct {
	kind string
	data []byte
}

// Size is the width and height of a sticker.
const Size = 512

// MaxBytes is the largest sticker this makes; WhatsApp may not show
// bigger ones.
const MaxBytes = 1 << 20

// maxPixels bounds the picture to decode (96 MB as RGBA).
const maxPixels = 24 << 20

// ErrTooLarge means the picture is too big to decode.
var ErrTooLarge = errors.New("sticker: the picture is too large")

// ErrAnimated means the picture is an animated WebP, which can't be
// decoded (see internal/webpanim).
var ErrAnimated = errors.New("sticker: the picture is animated")

// FromImage makes a sticker of a JPEG, PNG, GIF (its first frame) or still
// WebP picture, with text drawn on it.
func FromImage(data []byte, pack *Pack) ([]byte, error) {
	if animatedWebP(data) {
		return nil, ErrAnimated
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return nil, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	c, _ := canvas(src)
	argb := unpremultiply(c)

	var out []byte
	for quant := range uint(4) {
		out = insertMetadata(encodeWebP(argb, Size, Size, quant), pack)
		if len(out) <= MaxBytes {
			return out, nil
		}
	}

	return nil, errors.New("sticker: the picture doesn't fit in a sticker")
}

func FromVideo(data []byte, pack *Pack) ([]byte, error) {
	return nil, errors.New("sticker: video to sticker conversion is not implemented yet")
}

func canvas(img image.Image) (*image.RGBA, image.Rectangle) {
	b := img.Bounds()
	w, h := Size, Size
	if b.Dx() > b.Dy() {
		h = max(1, Size*b.Dy()/b.Dx())
	} else {
		w = max(1, Size*b.Dx()/b.Dy())
	}
	area := image.Rect(0, 0, w, h).Add(image.Pt((Size-w)/2, (Size-h)/2))
	c := image.NewRGBA(image.Rect(0, 0, Size, Size))
	if b.Dx() >= w && b.Dy() >= h {
		draw.Draw(c, area, photo.Shrink(img, w, h), image.Point{}, draw.Src)
	} else {
		xdraw.ApproxBiLinear.Scale(c, area, img, b, draw.Src, nil)
	}
	return c, area
}

// animatedWebP reports whether data is a WebP with animation.
func animatedWebP(data []byte) bool {
	// RIFF, size, WEBP, then a VP8X chunk whose flags say so.
	return len(data) >= 21 && string(data[0:4]) == "RIFF" && string(data[8:16]) == "WEBPVP8X" && data[20]&0x02 != 0
}

// unpremultiply returns the pixels of img as non-premultiplied ARGB, as
// WebP keeps them.
func unpremultiply(img *image.RGBA) []uint32 {
	b := img.Bounds()
	out := make([]uint32, 0, b.Dx()*b.Dy())
	for y := range b.Dy() {
		row := img.Pix[y*img.Stride : y*img.Stride+4*b.Dx()]
		for x := range b.Dx() {
			p := row[4*x : 4*x+4]
			r, g, bl, a := uint32(p[0]), uint32(p[1]), uint32(p[2]), uint32(p[3])
			if a != 0 && a != 0xff {
				r, g, bl = min(255, (r*255+a/2)/a), min(255, (g*255+a/2)/a), min(255, (bl*255+a/2)/a)
			}
			out = append(out, a<<24|r<<16|g<<8|bl)
		}
	}
	return out
}

func intFlag(enabled bool) *int {
	if !enabled {
		return nil
	}

	v := 1
	return &v
}

func createExif(pack *Pack) ([]byte, error) {
	meta := stickerMetadata{
		PackID:          "com.whatsapp.sticker_pack",
		PackName:        pack.Name,
		Publisher:       pack.Author,
		AndroidLink:     nil,
		IOSLink:         nil,
		AiSticker:       intFlag(pack.AI_STICKER),
		Emojis:          []string{""},
		UserCreatedPack: nil,
		AvatarSticker:   nil,
		Premium:         intFlag(pack.PREMIUM),
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	// TIFF little-endian header + one IFD entry.
	exif := []byte{
		0x49, 0x49, 0x2a, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x41, 0x57,
		0x07, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x16, 0x00, 0x00, 0x00,
	}

	binary.LittleEndian.PutUint32(exif[14:18], uint32(len(data)))
	exif = append(exif, data...)

	return exif, nil
}

func parseWebP(data []byte) ([]webpChunk, error) {
	if len(data) < 12 ||
		string(data[:4]) != "RIFF" ||
		string(data[8:12]) != "WEBP" {
		return nil, errors.New("invalid WebP header")
	}

	riffSize := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if riffSize != uint64(len(data)) {
		return nil, errors.New("invalid RIFF size")
	}

	var chunks []webpChunk

	for pos := 12; pos < len(data); {
		if len(data)-pos < 8 {
			return nil, errors.New("truncated WebP chunk header")
		}

		kind := string(data[pos : pos+4])
		size := uint64(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		start := uint64(pos) + 8
		end := start + size
		next := end + (size & 1)

		if end > uint64(len(data)) || next > uint64(len(data)) {
			return nil, errors.New("invalid WebP chunk size")
		}

		chunks = append(chunks, webpChunk{
			kind: kind,
			data: bytes.Clone(data[int(start):int(end)]),
		})

		pos = int(next)
	}

	return chunks, nil
}

func encodeChunk(chunk webpChunk) []byte {
	size := len(chunk.data)
	out := make([]byte, 8, 8+size+(size&1))

	copy(out[:4], chunk.kind)
	binary.LittleEndian.PutUint32(out[4:8], uint32(size))
	out = append(out, chunk.data...)

	if size&1 != 0 {
		out = append(out, 0)
	}

	return out
}

func webpDimensions(chunks []webpChunk) (uint32, uint32, bool, error) {
	var alpha bool

	for _, c := range chunks {
		switch c.kind {
		case "ALPH":
			alpha = true

		case "VP8L":
			if len(c.data) < 5 || c.data[0] != 0x2f {
				return 0, 0, false, errors.New("invalid VP8L chunk")
			}

			b := c.data
			w := 1 + uint32(b[1]) + (uint32(b[2]&0x3f) << 8)
			h := 1 + uint32(b[2]>>6) +
				(uint32(b[3]) << 2) +
				(uint32(b[4]&0x0f) << 10)

			alpha = alpha || (b[4]&0x10 != 0)
			return w, h, alpha, nil

		case "VP8 ":
			if len(c.data) < 10 ||
				c.data[3] != 0x9d ||
				c.data[4] != 0x01 ||
				c.data[5] != 0x2a {
				return 0, 0, false, errors.New("invalid VP8 frame")
			}

			w := binary.LittleEndian.Uint16(c.data[6:8]) & 0x3fff
			h := binary.LittleEndian.Uint16(c.data[8:10]) & 0x3fff

			if w == 0 || h == 0 {
				return 0, 0, false, errors.New("invalid VP8 dimensions")
			}

			return uint32(w), uint32(h), alpha, nil
		}
	}

	return 0, 0, false, errors.New("image dimensions not found")
}

func insertMetadata(sticker []byte, pack *Pack) []byte {
	if pack == nil {
		return sticker
	}

	exif, err := createExif(pack)
	if err != nil {
		return sticker
	}

	chunks, err := parseWebP(sticker)
	if err != nil {
		return sticker
	}

	var vp8x bool
	var hasAnimation bool

	filtered := make([]webpChunk, 0, len(chunks)+1)

	for _, c := range chunks {
		switch c.kind {
		case "EXIF":

			continue
		case "VP8X":
			if vp8x || len(c.data) != 10 {
				return sticker
			}
			vp8x = true
			c.data[0] |= 0x08
		case "ANIM", "ANMF":
			hasAnimation = true
		}

		filtered = append(filtered, c)
	}

	if !vp8x {
		if hasAnimation {
			return sticker
		}

		w, h, alpha, err := webpDimensions(filtered)
		if err != nil || w == 0 || h == 0 ||
			w > 1<<24 || h > 1<<24 {
			return sticker
		}

		flags := byte(0x08)
		if alpha {
			flags |= 0x10
		}

		vp8xData := make([]byte, 10)
		vp8xData[0] = flags

		// VP8X stores canvas dimensions minus one, in 24-bit LE.
		write24(vp8xData[4:7], w-1)
		write24(vp8xData[7:10], h-1)

		filtered = append(
			[]webpChunk{{kind: "VP8X", data: vp8xData}},
			filtered...,
		)
	}

	exifIndex := len(filtered)
	for i, c := range filtered {
		if c.kind == "XMP " {
			exifIndex = i
			break
		}
	}

	filtered = append(filtered, webpChunk{})
	copy(filtered[exifIndex+1:], filtered[exifIndex:])
	filtered[exifIndex] = webpChunk{
		kind: "EXIF",
		data: exif,
	}

	var out bytes.Buffer
	out.WriteString("RIFF")
	out.Write([]byte{0, 0, 0, 0})
	out.WriteString("WEBP")

	for _, c := range filtered {
		out.Write(encodeChunk(c))
	}

	result := out.Bytes()
	if uint64(len(result)-8) > uint64(^uint32(0)) {
		return sticker
	}

	binary.LittleEndian.PutUint32(
		result[4:8],
		uint32(len(result)-8),
	)

	return result
}

func write24(dst []byte, value uint32) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
}
