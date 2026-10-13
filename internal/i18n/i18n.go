// Package i18n provides the application's translation catalog.
package i18n

import (
	"embed"
	"fmt"

	"github.com/leonelquinteros/gotext"
)

//go:embed locales/*.po
var catalogs embed.FS

const DefaultLanguage = "es"

// Localizer owns the catalogs used by one UI instance.
type Localizer struct {
	language string
	catalogs map[string]map[string]string
}

// New loads the embedded catalogs and selects lang, falling back to Spanish.
func New(lang string) *Localizer {
	l := &Localizer{catalogs: make(map[string]map[string]string)}
	for _, language := range []string{"en", "es"} {
		data, err := catalogs.ReadFile("locales/" + language + ".po")
		if err != nil {
			panic("i18n: embedded catalog is missing: " + language)
		}
		po := gotext.NewPo()
		po.Parse(data)
		// Build the immutable lookup once, outside the frame loop. Keep only
		// strings; the parsed catalog and its translation metadata can be freed.
		entries := make(map[string]string)
		for id, translation := range po.GetDomain().GetTranslations() {
			if translation.IsTranslated() {
				entries[id] = translation.Get()
			}
		}
		l.catalogs[language] = entries
	}
	l.SetLanguage(lang)
	return l
}

// SetLanguage selects a supported language and returns the normalized code.
func (l *Localizer) SetLanguage(lang string) string {
	if _, ok := l.catalogs[lang]; !ok {
		lang = DefaultLanguage
	}
	l.language = lang
	return lang
}

func (l *Localizer) Language() string { return l.language }

// Text translates a string without formatting arguments.
func (l *Localizer) Text(msgid string) string {
	if msgid == "" {
		return ""
	}
	if translation, ok := l.catalogs[l.language][msgid]; ok {
		return translation
	}
	return msgid
}

func (l *Localizer) Textf(msgid string, args ...any) string {
	return fmt.Sprintf(l.Text(msgid), args...)
}
