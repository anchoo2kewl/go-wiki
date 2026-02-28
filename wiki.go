// Package gowiki provides an embeddable markdown wiki editor for Go web apps.
//
// It combines a rendering pipeline (blackfriday + pre/post filters), a toolbar
// with markdown shortcuts, a fullscreen split-view editor with live preview,
// and optional image upload support.
//
// Usage:
//
//	w := gowiki.New(gowiki.WithPreviewEndpoint("/wiki/preview"))
//	html := w.RenderContent("# Hello")
//	http.Handle("/wiki/preview", w.PreviewHandler())
//	http.Handle("/wiki/assets/", http.StripPrefix("/wiki/assets/", w.AssetHandler()))
package gowiki

import (
	"html/template"
	"net/http"

	"github.com/anchoo2kewl/go-wiki/editor"
	"github.com/anchoo2kewl/go-wiki/handler"
	"github.com/anchoo2kewl/go-wiki/render"
)

// Wiki is the top-level API wrapping the renderer and editor config.
type Wiki struct {
	Renderer *render.Renderer

	rendererOpts    render.RendererOptions
	classConfig     render.ClassConfig
	hasCustomClasses bool
	previewEndpoint string
	uploadEndpoint  string
	fullscreen      bool
}

// New creates a Wiki with the given options. Defaults: all rendering features
// enabled, fullscreen on, Tailwind classes, no upload endpoint.
func New(opts ...Option) *Wiki {
	w := &Wiki{
		rendererOpts:    render.DefaultOptions(),
		classConfig:     render.DefaultClassConfig(),
		previewEndpoint: "/wiki/preview",
		fullscreen:      true,
	}
	for _, o := range opts {
		o(w)
	}
	if w.hasCustomClasses {
		w.Renderer = render.NewRendererWithClasses(w.rendererOpts, w.classConfig)
	} else {
		w.Renderer = render.NewRenderer(w.rendererOpts)
	}
	return w
}

// RenderContent converts markdown to HTML using the full pipeline.
func (w *Wiki) RenderContent(content string) string {
	return w.Renderer.Render(content)
}

// RenderContentDebug returns HTML and intermediate pipeline stages.
func (w *Wiki) RenderContentDebug(content string) (string, map[string]string) {
	return w.Renderer.RenderWithDebug(content, true)
}

// PreviewHandler returns an http.HandlerFunc for live preview POST requests.
// The handler reads "content" from the form body and returns {"html":"..."}.
func (w *Wiki) PreviewHandler() http.HandlerFunc {
	h := handler.NewPreviewHandler(w.Renderer)
	return h.ServeHTTP
}

// AssetHandler returns an http.Handler that serves the editor's embedded
// CSS and JS files. Mount with http.StripPrefix if needed.
func (w *Wiki) AssetHandler() http.Handler {
	return handler.AssetHandler()
}

// EditorHTML returns a complete HTML fragment for the editor — toolbar,
// textarea, fullscreen overlay, inline CSS/JS — ready to embed in a page.
func (w *Wiki) EditorHTML(content string) (template.HTML, error) {
	cfg := editor.EditorConfig{
		PreviewEndpoint:  w.previewEndpoint,
		UploadEndpoint:   w.uploadEndpoint,
		EnableFullscreen: w.fullscreen,
		EnableImageUpload: w.uploadEndpoint != "",
		TextareaName:     "content",
		InitialContent:   content,
	}
	return editor.RenderEditor(cfg)
}
