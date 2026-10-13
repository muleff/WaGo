package androidbridge

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigureTemp must run before capture, decoding or uploads on Android.
func ConfigureTemp(dataDir string) error {
	dir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		for _, name := range []string{"TEMP", "TMP"} {
			if err := os.Setenv(name, dir); err != nil {
				return err
			}
		}
	}
	return os.Setenv("TMPDIR", dir)
}
