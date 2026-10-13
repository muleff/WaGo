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

func TestAndroidContactAndNavigationCopy(t *testing.T) {
	l := New("es")
	for _, key := range []string{"Updates", "Starred messages", "Notification settings", "Advanced chat privacy", "Media, links and docs", "Recording…", "Microphone access was denied. Enable it in Android settings."} {
		if got := l.Text(key); got == key {
			t.Errorf("missing Spanish: %s", key)
		}
	}
}

func TestSpanishMobileLinkingInstructions(t *testing.T) {
	l := New("es")
	for _, key := range []string{"Link your WhatsApp", "Your 8-character code", "Enter this code in WhatsApp on your phone: %s", "Country code", "Resend code"} {
		if l.Text(key) == key {
			t.Errorf("untranslated linking instruction: %s", key)
		}
	}
}

func TestEmptyTextDoesNotExposeCatalogHeader(t *testing.T) {
	if got := New("es").Text(""); got != "" {
		t.Fatalf("empty text exposed catalog: %q", got)
	}
}

func TestSpanishMobileContactActions(t *testing.T) {
	l := New("es")
	for _, key := range []string{"New chat", "New group", "New community", "Search name or number", "Contacts on WhatsApp", "Add group members"} {
		if got := l.Text(key); got == key {
			t.Errorf("untranslated mobile action %q", key)
		}
	}
}

func TestSpanishPresenceAndMobileMenus(t *testing.T) {
	l := New("es")
	for _, key := range []string{"online", "typing…", "last seen %s", "today at %s", "yesterday at %s", "%s at %s", "Photos & videos", "Status privacy", "Explore", "Add status", "Message", "Photo", "Draft: ", "Edited"} {
		if got := l.Text(key); got == key {
			t.Errorf("untranslated mobile copy %q", key)
		}
	}
}
