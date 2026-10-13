package i18n

import "testing"

func TestLocalizerLanguagesAndFallback(t *testing.T) {
	l := New("es")
	if got := l.Text("Language"); got != "Idioma" {
		t.Fatalf("Spanish translation = %q, want %q", got, "Idioma")
	}
	if got := l.Text("missing"); got != "missing" {
		t.Fatalf("missing translation = %q, want msgid", got)
	}

	if got := l.SetLanguage("unknown"); got != DefaultLanguage {
		t.Fatalf("unsupported language = %q, want %q", got, DefaultLanguage)
	}
	if got := l.Text("Language"); got != "Idioma" {
		t.Fatalf("Spanish fallback translation = %q, want %q", got, "Idioma")
	}
}
