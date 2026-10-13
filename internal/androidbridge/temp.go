package androidbridge

import (
	"os"
	"path/filepath"
)

// ConfigureTemp must run before capture, decoding or uploads on Android.
func ConfigureTemp(dataDir string) error {
	dir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.Setenv("TMPDIR", dir)
}
