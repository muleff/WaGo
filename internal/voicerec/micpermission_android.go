//go:build android

package voicerec

import (
	"context"
	"errors"
	"github.com/muleff/WaGo/internal/androidbridge"
)

func microphonePermission(ctx context.Context) error {
	err := androidbridge.RequestMicrophone(ctx)
	if errors.Is(err, androidbridge.ErrPermissionDenied) {
		return ErrMicrophoneAccess
	}
	return err
}
