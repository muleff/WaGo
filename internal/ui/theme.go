package ui

import (
	_ "embed"
	"image/color"

	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// Palette holds every color the UI uses. The dark values were sampled from
// WhatsApp Desktop (2025 design); the light ones follow the same structure.
type Palette struct {
	Frame, FrameText                             color.NRGBA // title bar and rail
	PanelBorder, Divider                         color.NRGBA
	Panel, ChatBg                                color.NRGBA
	Hover, Selected, Search                      color.NRGBA
	Chip, ChipBorder, ChipText                   color.NRGBA
	ChipActive, ChipActiveBorder, ChipActiveText color.NRGBA

	Text, TextSecondary, Icon, IconActive color.NRGBA
	IconStrong                            color.NRGBA // header, composer and menu buttons
	Green, OnGreen                        color.NRGBA
	RailActive, RailSeparator             color.NRGBA

	Doodle                       color.NRGBA
	BubbleIn, BubbleOut          color.NRGBA
	MetaIn, MetaOut, TickRead    color.NRGBA
	Link                         color.NRGBA // web links in messages
	TextOut, SecondaryOut        color.NRGBA
	QuoteIn, QuoteOut            color.NRGBA
	BubbleLine, BubbleButton     color.NRGBA // dividers and text of message buttons
	CodeBg                       color.NRGBA // behind `inline code`
	MentionPill                  color.NRGBA // behind mentions of you
	Selection                    color.NRGBA // behind selected text
	DateChip, DateChipText       color.NRGBA
	Encryption, EncryptionText   color.NRGBA
	Composer, ComposerHint       color.NRGBA
	GroupAvatar, GroupAvatarIcon color.NRGBA
	UserAvatar, UserAvatarIcon   color.NRGBA
	// AvatarBgs and AvatarFgs color the initial of someone without a
	// picture, chosen by their ID; GeneralAvatar is behind a community's
	// General chat's speech bubble.
	AvatarBgs, AvatarFgs     []color.NRGBA
	GeneralAvatar            color.NRGBA
	Banner, BannerText, QRFg color.NRGBA
	Menu, MenuHover, Shadow  color.NRGBA
	CloseHover               color.NRGBA
	Senders                  []color.NRGBA

	EmptyIcon                        color.NRGBA // big glyphs of empty panes
	RowHover                         color.NRGBA // settings and info list rows
	Danger, DangerSoft               color.NRGBA // "Log out"; "Exit group" and friends
	RingViewed                       color.NRGBA // status ring once seen
	Verified                         color.NRGBA
	AnnounceBg, AnnounceIcon         color.NRGBA // community announcements tile
	ChannelAvatar, ChannelAvatarIcon color.NRGBA // channel without a picture
	StatusBg                         color.NRGBA // status viewer backdrop

	Popup, PopupBorder, PopupHover color.NRGBA // context menus and the reaction bar
	PopupDivider, PopupSub         color.NRGBA
	Picker, PickerTab              color.NRGBA // emoji picker panel; its active tab
	PickerTabBorder                color.NRGBA
	Viewer, ViewerDivider          color.NRGBA // media viewer
	ViewerThumb, ViewerArrow       color.NRGBA
	Scrim                          color.NRGBA // behind dialogs
	Dialog                         color.NRGBA
	Toast, ToastText               color.NRGBA
}

type accentColor struct {
	name string
	hex  uint32
}

var accentColors = []accentColor{
	{name: "Green", hex: 0x1daa61},
	{name: "Emerald", hex: 0x10a36a},
	{name: "Mint", hex: 0x45c49a},

	{name: "Blue", hex: 0x3478f6},
	{name: "Sky", hex: 0x3aa8e8},
	{name: "Azure", hex: 0x1677c8},

	{name: "Purple", hex: 0x8e5bd8},
	{name: "Violet", hex: 0x6f42c1},
	{name: "Lavender", hex: 0xa678d6},

	{name: "Orange", hex: 0xf08c2e},
	{name: "Amber", hex: 0xe6a21a},
	{name: "Coral", hex: 0xe87855},

	{name: "Red", hex: 0xe0533d},
	{name: "Crimson", hex: 0xc93c50},
	{name: "Rose", hex: 0xd94f70},

	{name: "Pink", hex: 0xe04e9d},
	{name: "Magenta", hex: 0xc63fa3},
	{name: "Blush", hex: 0xd875a8},

	{name: "Teal", hex: 0x1f8a7d},
	{name: "Turquoise", hex: 0x20a99a},
	{name: "Seafoam", hex: 0x52bfae},

	{name: "Lime", hex: 0x9abf00},
	{name: "Olive", hex: 0x7f9518},
	{name: "Chartreuse", hex: 0xb2c92e},

	{name: "Yellow", hex: 0xf5c200},
	{name: "Gold", hex: 0xd9a514},
	{name: "Lemon", hex: 0xe8d33f},

	{name: "Cyan", hex: 0x00bcd4},
	{name: "Aqua", hex: 0x20c4d8},
	{name: "Cerulean", hex: 0x268fb5},

	{name: "Indigo", hex: 0x3f51b5},
	{name: "Sapphire", hex: 0x3155a6},
	{name: "Periwinkle", hex: 0x6574c7},

	{name: "Brown", hex: 0x795548},
	{name: "Mocha", hex: 0x8a6652},
	{name: "Copper", hex: 0xa56842},

	{name: "Gray", hex: 0x9e9e9e},
	{name: "Slate", hex: 0x687782},
	{name: "Silver", hex: 0xb0b7bd},
}

// markColors are what the photo editor draws with, the same in both
// themes since they end up in the photo.
var markColors = rgbs(0xffffff, 0x000000, 0xff3b30, 0xff9500, 0xffcc00, 0x34c759, 0x007aff, 0xaf52de)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

func argb(c uint32, a uint8) color.NRGBA {
	col := rgb(c)
	col.A = a
	return col
}

func rgbs(cs ...uint32) []color.NRGBA {
	out := make([]color.NRGBA, len(cs))
	for i, c := range cs {
		out[i] = rgb(c)
	}
	return out
}

func accentPalette(base *Palette, dark bool, name string) *Palette {
	if name == "" || name == accentColors[0].name {
		return base
	}
	var accent uint32
	for _, c := range accentColors {
		if c.name == name {
			accent = c.hex
			break
		}
	}
	if accent == 0 {
		return base
	}
	q := *base
	h := rgb(accent)
	q.Green = h
	q.Link = h
	q.OnGreen = rgb(0xffffff)
	if dark {
		q.BubbleOut = mix(rgb(0x1d1f1f), h, 0.42)
		q.QuoteOut = mix(q.BubbleOut, rgb(0x000000), 0.2)
		q.MetaOut = mix(q.BubbleOut, rgb(0xffffff), 0.62)
		q.SecondaryOut = q.MetaOut
		q.ChipActive = mix(base.ChipActive, h, 0.28)
		q.ChipActiveBorder = mix(base.ChipActiveBorder, h, 0.42)
	} else {
		q.BubbleOut = mix(rgb(0xffffff), h, 0.24)
		q.QuoteOut = mix(q.BubbleOut, h, 0.14)
		q.MetaOut = mix(q.BubbleOut, rgb(0x000000), 0.48)
		q.SecondaryOut = q.MetaOut
		q.ChipActive = mix(rgb(0xffffff), h, 0.18)
		q.ChipActiveBorder = mix(base.ChipActiveBorder, h, 0.42)
	}
	q.ChipActiveText = h
	q.MentionPill = argb(accent, 0x24)
	return &q
}

var darkPalette = Palette{
	Frame: rgb(0x1d1f1e), FrameText: rgb(0xfafafa),
	PanelBorder: rgb(0x2c2c2c), Divider: rgb(0x282828),
	Panel: rgb(0x171717), ChatBg: rgb(0x171717),
	Hover: rgb(0x2e2f2f), Selected: rgb(0x2e2f2f), Search: rgb(0x2e2f2f),
	Chip: rgb(0x171717), ChipBorder: rgb(0x2e2f2f), ChipText: rgb(0xa5a7a9),
	ChipActive: rgb(0x1a342b), ChipActiveBorder: rgb(0x2c4a3d), ChipActiveText: rgb(0xd9fdd3),

	Text: rgb(0xfafafa), TextSecondary: rgb(0xa5a7a8), Icon: rgb(0xaaacab), IconActive: rgb(0xfafafa),
	IconStrong: rgb(0xffffff),
	Green:      rgb(0x5dbf6e), OnGreen: rgb(0x0a1a0e),
	RailActive: rgb(0x333535), RailSeparator: rgb(0x2f3131),

	Doodle:   rgb(0x292c2b),
	BubbleIn: rgb(0x242626), BubbleOut: rgb(0x254d39),
	MetaIn: rgb(0xaaacab), MetaOut: rgb(0xa3c0b0), TickRead: rgb(0x53bdeb),
	Link:    rgb(0x5dbf6e),
	TextOut: rgb(0xfafafa), SecondaryOut: rgb(0xa6c2b4),
	QuoteIn: rgb(0x1d1f1f), QuoteOut: rgb(0x1f4232),
	DateChip: rgb(0x1d1f1e), DateChipText: rgb(0xa6a8a8),
	Encryption: rgb(0x1d1f1e), EncryptionText: rgb(0xe3c77b),
	Composer: rgb(0x242626), ComposerHint: rgb(0xabadac),
	GroupAvatar: rgb(0x102540), GroupAvatarIcon: rgb(0x70adff),
	UserAvatar: rgb(0x303434), UserAvatarIcon: rgb(0xa6abad),
	AvatarBgs:     rgbs(0x242346, 0x102540, 0x0f3330, 0x16321d, 0x3a2e10, 0x3d2414, 0x3d1a2b),
	AvatarFgs:     rgbs(0xa598f3, 0x70adff, 0x5fcfbf, 0x6fd38a, 0xefc152, 0xf5a172, 0xf18db5),
	GeneralAvatar: rgb(0xd1d8da),
	Banner:        rgb(0x2e2f2f), BannerText: rgb(0xd0d2d2), QRFg: rgb(0x122e31),
	Menu: rgb(0x242626), MenuHover: rgb(0x2e2f2f), Shadow: argb(0x000000, 0x60),
	CloseHover: rgb(0xc42b1c),
	Senders:    rgbs(0xcca48f, 0x8fb8e8, 0xe6a1b8, 0x86c9a8, 0xdcc27a, 0xb1a3e6, 0xe89b7f, 0x7fc3d6, 0xc7b7a0),

	EmptyIcon: rgb(0x454545), RowHover: rgb(0x1d1f1f),
	Danger: rgb(0xe85d65), DangerSoft: rgb(0xed9ea5),
	RingViewed: rgb(0x414141), Verified: rgb(0x3c79e6),
	AnnounceBg: rgb(0x342c21), AnnounceIcon: rgb(0xf6d78b),
	ChannelAvatar: rgb(0x32281e), ChannelAvatarIcon: rgb(0xd3a887),
	StatusBg: rgb(0x0f0f0f),

	Popup: rgb(0x161717), PopupBorder: rgb(0x2b2c2c), PopupHover: rgb(0x242626),
	PopupDivider: rgb(0x292a2a), PopupSub: rgb(0x969696),
	Picker: rgb(0x1d1f1f), PickerTab: rgb(0x242626), PickerTabBorder: rgb(0x343636),
	Viewer: rgb(0x171717), ViewerDivider: rgb(0x1f1f1f),
	ViewerThumb: rgb(0x242625), ViewerArrow: rgb(0x0e0e0e),
	Scrim: argb(0x000000, 0x99), Dialog: rgb(0x1d1f1f),
	Toast: rgb(0x2e2f2f), ToastText: rgb(0xfafafa),
	BubbleLine: argb(0xffffff, 0x14), BubbleButton: rgb(0x53bdeb),
	CodeBg:      argb(0xffffff, 0x12),
	MentionPill: argb(0x21c063, 0x24),
	Selection:   rgb(0x1e3a9e),
}

var lightPalette = Palette{
	Frame: rgb(0xf7f5f3), FrameText: rgb(0x0a0a0a),
	PanelBorder: rgb(0xe3e0dc), Divider: rgb(0xe9edef),
	Panel: rgb(0xffffff), ChatBg: rgb(0xf5f1eb),
	Hover: rgb(0xf5f6f6), Selected: rgb(0xf0f2f5), Search: rgb(0xf6f5f4),
	Chip: rgb(0xffffff), ChipBorder: rgb(0xe3e0dc), ChipText: rgb(0x5e6468),
	ChipActive: rgb(0xd9fdd3), ChipActiveBorder: rgb(0xc3eebc), ChipActiveText: rgb(0x15603e),

	Text: rgb(0x0a0a0a), TextSecondary: rgb(0x667781), Icon: rgb(0x54656f), IconActive: rgb(0x0a0a0a),
	IconStrong: rgb(0x0a0a0a),
	Green:      rgb(0x1daa61), OnGreen: rgb(0xffffff),
	RailActive: rgb(0xe8e6e3), RailSeparator: rgb(0xe3e0dc),

	Doodle:   rgb(0xe8e1d8),
	BubbleIn: rgb(0xffffff), BubbleOut: rgb(0xd9fdd3),
	MetaIn: rgb(0x667781), MetaOut: rgb(0x5b7a66), TickRead: rgb(0x53bdeb),
	Link:    rgb(0x027eb5),
	TextOut: rgb(0x0a0a0a), SecondaryOut: rgb(0x587a64),
	QuoteIn: rgb(0xf5f6f6), QuoteOut: rgb(0xd1f4cc),
	DateChip: rgb(0xffffff), DateChipText: rgb(0x54656f),
	Encryption: rgb(0xffeecd), EncryptionText: rgb(0x54656f),
	Composer: rgb(0xffffff), ComposerHint: rgb(0x667781),
	GroupAvatar: rgb(0xdbe7fb), GroupAvatarIcon: rgb(0x3778e5),
	UserAvatar: rgb(0xe8ebed), UserAvatarIcon: rgb(0x8a9499),
	AvatarBgs:     rgbs(0xe8e5fc, 0xdbe7fb, 0xd5f2ee, 0xd9f2df, 0xf8edcf, 0xfbe3d4, 0xfadbe8),
	AvatarFgs:     rgbs(0x5e4fc8, 0x3778e5, 0x1f8a7d, 0x2b8a45, 0x9a6c00, 0xb3561f, 0xb43d72),
	GeneralAvatar: rgb(0xdfe5e7),
	Banner:        rgb(0xfff4c5), BannerText: rgb(0x54656f), QRFg: rgb(0x122e31),
	Menu: rgb(0xffffff), MenuHover: rgb(0xf5f6f6), Shadow: argb(0x0b141a, 0x30),
	CloseHover: rgb(0xc42b1c),
	Senders:    rgbs(0x1f7aec, 0xe542a3, 0x02a698, 0xc85a00, 0x7f66ff, 0xd62f45, 0x029d00, 0x0e8a94, 0xa4661f),

	EmptyIcon: rgb(0xc4c9cc), RowHover: rgb(0xf5f6f6),
	Danger: rgb(0xea0038), DangerSoft: rgb(0xd4314a),
	RingViewed: rgb(0xc3c9cc), Verified: rgb(0x2a7de1),
	AnnounceBg: rgb(0xfdf1d8), AnnounceIcon: rgb(0xc58a13),
	ChannelAvatar: rgb(0xf6e7da), ChannelAvatarIcon: rgb(0xa86f45),
	StatusBg: rgb(0x0f0f0f),

	Popup: rgb(0xffffff), PopupBorder: rgb(0xe9edef), PopupHover: rgb(0xf5f6f6),
	PopupDivider: rgb(0xe9edef), PopupSub: rgb(0x667781),
	Picker: rgb(0xffffff), PickerTab: rgb(0xf0f2f5), PickerTabBorder: rgb(0xe3e0dc),
	Viewer: rgb(0xffffff), ViewerDivider: rgb(0xe9edef),
	ViewerThumb: rgb(0xf0f2f5), ViewerArrow: rgb(0xf0f2f5),
	Scrim: argb(0xffffff, 0xb0), Dialog: rgb(0xffffff),
	Toast: rgb(0x3b4a54), ToastText: rgb(0xffffff),
	BubbleLine: argb(0x000000, 0x14), BubbleButton: rgb(0x027eb5),
	CodeBg:      argb(0x000000, 0x0f),
	MentionPill: argb(0x1daa61, 0x22),
	Selection:   rgb(0xb3d4fc),
}

func hashIndex(s string, n int) int {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h % uint32(n))
}

// NotoColorEmoji is a bitmap (CBDT) color emoji font. Gio can't draw the COLR
// glyphs of Windows' Segoe UI Emoji in color, so it's bundled.
//
//go:embed fonts/NotoColorEmoji.ttf
var notoColorEmoji []byte

// Typeface prefers Segoe UI (what WhatsApp Desktop uses on Windows), color
// emoji from Noto, then common system UI fonts and the bundled Go fonts.
//
// MS Gothic covers Japanese and most Chinese text in a single file for
// every weight. Without it the fallback picks Yu Gothic, whose weights are
// separate files that each cost ~40 MB of heap once loaded (the whole
// glyph table is parsed), so a bold name plus a regular message would load
// two of them.
const typeface font.Typeface = "Segoe UI, Roboto, Helvetica Neue, Noto Sans, Go, " + emojiTypeface + ", Segoe UI Symbol, MS Gothic, Cambria Math, sans-serif"

func newTheme() *material.Theme {
	th := material.NewTheme()
	// System fonts are enabled by default and cover other scripts.
	th.Shaper = text.NewShaper(text.WithCollection(bundledFonts()))
	th.Face = typeface
	th.TextSize = 14
	return th
}
