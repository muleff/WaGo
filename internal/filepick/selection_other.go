//go:build !android

package filepick

import "path/filepath"

func openSelection(title string, multiple bool, filters []Filter) ([]Selection, error) {
	paths, err := open(title, multiple, filters)
	out := make([]Selection, 0, len(paths))
	for _, p := range paths {
		out = append(out, Selection{Path: p, Name: filepath.Base(p)})
	}
	return out, err
}
