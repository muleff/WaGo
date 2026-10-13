//go:build !windows && !android

package voicerec

import "context"

func microphonePermission(ctx context.Context) error { return nil }
