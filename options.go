package gowiki

import "github.com/anchoo2kewl/go-wiki/render"

// Option is a functional option for configuring a Wiki instance.
type Option func(*Wiki)

// WithRendererOptions sets custom renderer options.
func WithRendererOptions(opt render.RendererOptions) Option {
	return func(w *Wiki) {
		w.rendererOpts = opt
	}
}

// WithClassConfig sets custom CSS classes for the renderer.
func WithClassConfig(cc render.ClassConfig) Option {
	return func(w *Wiki) {
		w.classConfig = cc
		w.hasCustomClasses = true
	}
}

// WithPreviewEndpoint sets the URL path used for live preview POSTs.
func WithPreviewEndpoint(endpoint string) Option {
	return func(w *Wiki) {
		w.previewEndpoint = endpoint
	}
}

// WithUploadEndpoint sets the URL for image uploads.
func WithUploadEndpoint(endpoint string) Option {
	return func(w *Wiki) {
		w.uploadEndpoint = endpoint
	}
}

// WithFullscreen enables or disables the fullscreen editor.
func WithFullscreen(enabled bool) Option {
	return func(w *Wiki) {
		w.fullscreen = enabled
	}
}

// WithDrawBasePath enables go-draw embed support and sets the URL prefix
// for go-draw routes (e.g. "/draw"). When set, [draw:id] and [draw:id:edit]
// shortcodes in markdown are rendered as resizable go-draw iframe embeds.
func WithDrawBasePath(basePath string) Option {
	return func(w *Wiki) {
		w.rendererOpts.EnableDrawEmbeds = true
		w.rendererOpts.DrawBasePath = basePath
	}
}
