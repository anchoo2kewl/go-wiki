package handler

import (
	"net/http"

	"github.com/anchoo2kewl/go-wiki/editor"
)

// AssetHandler returns an http.Handler that serves the editor's embedded
// CSS and JS files. Mount it at a path like "/go-wiki/assets/".
func AssetHandler() http.Handler {
	return http.FileServer(http.FS(editor.Assets()))
}
