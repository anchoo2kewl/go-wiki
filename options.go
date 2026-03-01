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

// WithImageListEndpoint sets the URL for the image list/browse API.
// When set, an "Images" button appears in the toolbar that opens a modal
// browser showing uploaded images.
func WithImageListEndpoint(endpoint string) Option {
	return func(w *Wiki) {
		w.imageListEndpoint = endpoint
	}
}

// WithEnableMore adds a "More" button to the toolbar that inserts
// a <more--> marker, commonly used to separate post excerpts.
func WithEnableMore(enabled bool) Option {
	return func(w *Wiki) {
		w.enableMore = enabled
	}
}

// WithImageMetadataEndpoint sets the URL for CRUD operations on image metadata.
// When set, the image manager tracks uploaded images with alt text and captions.
func WithImageMetadataEndpoint(endpoint string) Option {
	return func(w *Wiki) {
		w.imageMetadataEndpoint = endpoint
	}
}

// WithCloudinarySignatureEndpoint sets the URL for generating Cloudinary upload signatures.
// When set, the editor supports direct browser-to-Cloudinary uploads.
func WithCloudinarySignatureEndpoint(endpoint string) Option {
	return func(w *Wiki) {
		w.cloudinarySignatureEndpoint = endpoint
	}
}

// WithCloudinaryCloudName sets the Cloudinary cloud name for direct uploads.
func WithCloudinaryCloudName(cloudName string) Option {
	return func(w *Wiki) {
		w.cloudinaryCloudName = cloudName
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
