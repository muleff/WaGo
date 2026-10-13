//go:build !windows

package voicerec

func microphonePermission() error { return nil }
