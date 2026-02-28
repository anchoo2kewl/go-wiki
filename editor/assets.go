package editor

import (
	"embed"
	"io/fs"
)

//go:embed css/*.css js/*.js templates/*.gohtml
var embeddedFS embed.FS

// Assets returns the embedded filesystem containing CSS, JS, and templates.
// Use this with http.FileServer or to read individual files.
func Assets() fs.FS {
	return embeddedFS
}
