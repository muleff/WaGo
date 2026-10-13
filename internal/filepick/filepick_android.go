//go:build android

package filepick

import (
	"context"
	"github.com/muleff/WaGo/internal/androidbridge"
	"mime"
	"strings"
)

func openSelection(title string, multiple bool, filters []Filter) ([]Selection, error) {
	var types []string
	for _, f := range filters {
		for _, ext := range f.Exts {
			if m := mime.TypeByExtension("." + ext); m != "" {
				types = append(types, strings.Split(m, ";")[0])
			}
		}
	}
	files, err := androidbridge.PickFiles(context.Background(), multiple, types)
	out := make([]Selection, 0, len(files))
	for _, f := range files {
		out = append(out, Selection{Path: f.Path, Name: f.Name, MIME: f.MIME})
	}
	return out, err
}
func open(title string, multiple bool, filters []Filter) ([]string, error) {
	files, err := openSelection(title, multiple, filters)
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out, err
}
