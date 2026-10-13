//go:build !android

package ui

func registerMobileLifecycle(h *host) func() { return func() {} }
