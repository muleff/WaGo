//go:build android

package ui

import (
	"github.com/muleff/WaGo/internal/androidbridge"
	"sync"
)

var androidOpenState struct {
	sync.Mutex
	notify func()
	errors chan error
}

func openAndroidTarget(target string) bool {
	androidOpenState.Lock()
	notify := androidOpenState.notify
	if androidOpenState.errors == nil {
		androidOpenState.errors = make(chan error, 4)
	}
	errors := androidOpenState.errors
	androidOpenState.Unlock()
	go func() {
		if err := androidbridge.OpenTarget(target); err != nil {
			select {
			case errors <- err:
			default:
			}
			if notify != nil {
				notify()
			}
		}
	}()
	return true
}
func (u *UI) pollSystemOpen() {
	androidOpenState.Lock()
	androidOpenState.notify = u.images.invalidate
	errors := androidOpenState.errors
	androidOpenState.Unlock()
	select {
	case <-errors:
		u.toast(u.locale.Text("No app is available to open this link."))
	default:
	}
}
