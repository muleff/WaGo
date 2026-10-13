package filepick

// Selection retains SAF metadata separately from its private local filename.
type Selection struct{ Path, Name, MIME string }

func OpenSelection(title string, multiple bool, filters ...Filter) ([]Selection, error) {
	return openSelection(title, multiple, filters)
}
