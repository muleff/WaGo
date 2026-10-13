//go:build !android

package wa

import "errors"

func openAndroidFile(path string) error { return errors.New("Android media opening is unavailable") }
