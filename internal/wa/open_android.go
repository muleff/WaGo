//go:build android

package wa

import (
	"github.com/muleff/WaGo/internal/androidbridge"
)

func openAndroidFile(path string) error {
	typ := externalMediaType(path)
	return androidbridge.OpenMedia(path, typ)
}
