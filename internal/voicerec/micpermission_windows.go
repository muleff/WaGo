package voicerec

import (
	"context"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Windows privacy settings have a separate switch for unpackaged desktop
// applications. Detect an explicit denial before attempting DirectShow
// enumeration, which can otherwise look like "no microphone".
func microphonePermission(ctx context.Context) error {
	const consentStore = `SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\microphone`
	for _, path := range []string{consentStore, consentStore + `\NonPackaged`} {
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		value, _, readErr := key.GetStringValue("Value")
		_ = key.Close()
		if readErr == nil && strings.EqualFold(value, "Deny") {
			return ErrMicrophoneAccess
		}
	}
	return nil
}
