package androidbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemporaryFilesUseAppPrivateDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TMPDIR", "/data/local/tmp")
	t.Setenv("TEMP", os.Getenv("TEMP"))
	t.Setenv("TMP", os.Getenv("TMP"))
	if err := ConfigureTemp(root); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp("", "voice-*.ogg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if !strings.HasPrefix(f.Name(), filepath.Join(root, "tmp")+string(os.PathSeparator)) {
		t.Fatalf("temp outside app: %s", f.Name())
	}
}
