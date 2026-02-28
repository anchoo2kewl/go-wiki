package handler

import (
	"encoding/json"
	"net/http"

	"github.com/anchoo2kewl/go-wiki/render"
)

// PreviewHandler serves live-preview requests.
// It accepts POST with a "content" form field and returns {"html":"..."} JSON.
type PreviewHandler struct {
	Renderer *render.Renderer
}

// NewPreviewHandler creates a PreviewHandler wrapping the given Renderer.
func NewPreviewHandler(r *render.Renderer) *PreviewHandler {
	return &PreviewHandler{Renderer: r}
}

// ServeHTTP implements http.Handler.
func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	html := h.Renderer.Render(content)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"html": html})
}
