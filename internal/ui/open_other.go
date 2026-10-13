//go:build !android

package ui

func openAndroidTarget(target string) bool { return false }
func (u *UI) pollSystemOpen()              {}
