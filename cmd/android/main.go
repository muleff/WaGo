// Command android starts OpenWA on Android with the live WhatsApp backend.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"gioui.org/app"

	"github.com/skidy89/openWA/internal/accounts"
	"github.com/skidy89/openWA/internal/model"
	"github.com/skidy89/openWA/internal/ui"
	"github.com/skidy89/openWA/internal/wa"
)

func main() {
	debug.SetGCPercent(50)

	// Gio resolves Android's private Context.getFilesDir() directory.
	// This must be called from main, after Android has initialized Gio.
	dataDir, err := app.DataDir()
	if err != nil {
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
