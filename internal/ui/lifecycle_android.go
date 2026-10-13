//go:build android

package ui

import "github.com/muleff/WaGo/internal/androidbridge"

func registerMobileLifecycle(h *host) func() {
	androidbridge.SetForeground(func() { h.request(request{kind: reqResume}) })
	return func() { androidbridge.SetForeground(nil) }
}
