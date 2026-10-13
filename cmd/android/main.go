// Command android starts OpenWA on Android with the live WhatsApp backend.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	_ "time/tzdata"

	"gioui.org/app"

	// Permission packages add their AndroidManifest entries at packaging
	// time (voice notes need RECORD_AUDIO, attachments need storage,
	// profile photos need the camera). AndroidBridge requests microphone
	// permission on first use. SAF selection does not need storage permission.
	_ "gioui.org/app/permission/camera"
	_ "gioui.org/app/permission/microphone"

	"github.com/muleff/WaGo/internal/accounts"
	"github.com/muleff/WaGo/internal/androidbridge"
	"github.com/muleff/WaGo/internal/model"
	"github.com/muleff/WaGo/internal/ui"
	"github.com/muleff/WaGo/internal/wa"
)

func main() {
	debug.SetGCPercent(50)
	if err := androidbridge.ConfigureLocalTime(); err != nil {
		log.Print(err)
	}

	// Gio resolves Android's private Context.getFilesDir() directory.
	// This must be called from main, after Android has initialized Gio.
	dataDir, err := app.DataDir()
	if err != nil {
		log.Fatal(err)
	}
	if err := androidbridge.ConfigureTemp(dataDir); err != nil {
		log.Fatal(err)
	}
	go run(filepath.Join(dataDir, "WazzapClients"))
	app.Main()
}

// run initializes the database off the main thread, which Gio needs for Android.
func run(dataDir string) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatal(err)
	}

	list := accounts.Load(dataDir)

	backend, err := wa.Open(list.Path(list.Active), false, true)
	if err != nil {
		log.Fatal(err)
	}

	options := ui.Options{
		Mobile: true,
		Window: []app.Option{
			app.Title("OpenWA"),
		},
		NotifyDir: filepath.Join(dataDir, "notifications"),
		Accounts:  list,
		Open: func(dir string) (model.Backend, error) {
			return wa.Open(dir, false, true)
		},
	}

	if err := ui.Run(backend, options); err != nil {
		log.Print(err)
		os.Exit(1)
	}
	os.Exit(0)
}
