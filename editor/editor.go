package editor

import (
	"bytes"
	"html/template"
)

// EditorConfig controls what the rendered editor includes.
type EditorConfig struct {
	PreviewEndpoint             string // required: URL for live preview POST (e.g. "/wiki/preview")
	UploadEndpoint              string // optional: if empty, upload buttons are hidden
	ImageListEndpoint           string // optional: if set, an "Images" browser button is shown
	ImageMetadataEndpoint       string // optional: CRUD endpoint for image metadata
	CloudinarySignatureEndpoint string // optional: endpoint for generating Cloudinary upload signatures
	CloudinaryCloudName         string // optional: Cloudinary cloud name for direct browser uploads
	EnableFullscreen            bool   // default: true
	EnableImageUpload           bool   // default: false
	EnableMore                  bool   // default: false; if true, a "More" button inserts <more-->
	EnableAnnotations           bool   // default: false; if true, annotations.js is inlined (exposes window.GoWikiAnnotations)
	DarkMode                    bool   // default: true (consumer controls via CSS class on <body>)
	TextareaName                string // form field name, default: "content"
	InitialContent              string // pre-filled content for the textarea
	ExtraCSS                    string // additional CSS to inject
	ExtraJS                     string // additional JS to inject
	DrawBasePath                string // optional: if set, a "Draw" button is shown (e.g. "/draw")
}

// DefaultEditorConfig returns a config with sensible defaults.
func DefaultEditorConfig(previewEndpoint string) EditorConfig {
	return EditorConfig{
		PreviewEndpoint:  previewEndpoint,
		EnableFullscreen: true,
		DarkMode:         true,
		TextareaName:     "content",
	}
}

// templateData is the data passed to Go templates.
type templateData struct {
	TextareaID                  string
	TextareaName                string
	InitialContent              string
	EnableFullscreen            bool
	EnableUpload                bool
	PreviewEndpoint             string
	UploadEndpoint              string
	ImageListEndpoint           string
	ImageMetadataEndpoint       string
	CloudinarySignatureEndpoint string
	CloudinaryCloudName         string
	DrawBasePath                string
	EnableMore                  bool
}

// RenderEditor returns the complete editor HTML fragment — toolbar, textarea,
// fullscreen overlay, config script, CSS, and JS — ready to drop into a page.
func RenderEditor(cfg EditorConfig) (template.HTML, error) {
	if cfg.TextareaName == "" {
		cfg.TextareaName = "content"
	}

	data := templateData{
		TextareaID:                  "gw-editor",
		TextareaName:                cfg.TextareaName,
		InitialContent:              cfg.InitialContent,
		EnableFullscreen:            cfg.EnableFullscreen,
		EnableUpload:                cfg.EnableImageUpload && cfg.UploadEndpoint != "",
		PreviewEndpoint:             cfg.PreviewEndpoint,
		UploadEndpoint:              cfg.UploadEndpoint,
		ImageListEndpoint:           cfg.ImageListEndpoint,
		ImageMetadataEndpoint:       cfg.ImageMetadataEndpoint,
		CloudinarySignatureEndpoint: cfg.CloudinarySignatureEndpoint,
		CloudinaryCloudName:         cfg.CloudinaryCloudName,
		DrawBasePath:                cfg.DrawBasePath,
		EnableMore:                  cfg.EnableMore,
	}

	// Parse embedded templates
	tmpl, err := template.ParseFS(embeddedFS, "templates/editor.gohtml", "templates/fullscreen.gohtml")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	// Inline CSS
	buf.WriteString("<style>\n")
	if css, err := readEmbedded("css/editor.css"); err == nil {
		buf.Write(css)
	}
	if cfg.EnableFullscreen {
		if css, err := readEmbedded("css/fullscreen.css"); err == nil {
			buf.WriteString("\n")
			buf.Write(css)
		}
	}
	if cfg.ExtraCSS != "" {
		buf.WriteString("\n")
		buf.WriteString(cfg.ExtraCSS)
	}
	buf.WriteString("\n</style>\n")

	// Config script
	buf.WriteString(`<script>window.__goWikiConfig = {`)
	buf.WriteString(`previewEndpoint: "` + template.JSEscapeString(cfg.PreviewEndpoint) + `"`)
	buf.WriteString(`, textareaId: "gw-editor"`)
	buf.WriteString(`, previewId: "gw-preview-content"`)
	if cfg.UploadEndpoint != "" {
		buf.WriteString(`, uploadEndpoint: "` + template.JSEscapeString(cfg.UploadEndpoint) + `"`)
	}
	if cfg.ImageListEndpoint != "" {
		buf.WriteString(`, imageListEndpoint: "` + template.JSEscapeString(cfg.ImageListEndpoint) + `"`)
	}
	if cfg.ImageMetadataEndpoint != "" {
		buf.WriteString(`, imageMetadataEndpoint: "` + template.JSEscapeString(cfg.ImageMetadataEndpoint) + `"`)
	}
	if cfg.CloudinarySignatureEndpoint != "" {
		buf.WriteString(`, cloudinarySignatureEndpoint: "` + template.JSEscapeString(cfg.CloudinarySignatureEndpoint) + `"`)
	}
	if cfg.CloudinaryCloudName != "" {
		buf.WriteString(`, cloudinaryCloudName: "` + template.JSEscapeString(cfg.CloudinaryCloudName) + `"`)
	}
	if cfg.DrawBasePath != "" {
		buf.WriteString(`, drawBasePath: "` + template.JSEscapeString(cfg.DrawBasePath) + `"`)
	}
	if cfg.EnableMore {
		buf.WriteString(`, enableMore: true`)
	}
	buf.WriteString(`};</script>`)
	buf.WriteString("\n")

	// Render editor template
	if err := tmpl.ExecuteTemplate(&buf, "editor.gohtml", data); err != nil {
		return "", err
	}

	// Render fullscreen template
	if cfg.EnableFullscreen {
		if err := tmpl.ExecuteTemplate(&buf, "fullscreen.gohtml", data); err != nil {
			return "", err
		}
	}

	// Inline JS
	buf.WriteString("\n<script>\n")
	if js, err := readEmbedded("js/editor-core.js"); err == nil {
		buf.Write(js)
	}
	if cfg.EnableFullscreen {
		if js, err := readEmbedded("js/fullscreen.js"); err == nil {
			buf.WriteString("\n")
			buf.Write(js)
		}
	}
	if js, err := readEmbedded("js/keyboard.js"); err == nil {
		buf.WriteString("\n")
		buf.Write(js)
	}
	if cfg.EnableAnnotations {
		if js, err := readEmbedded("js/annotations.js"); err == nil {
			buf.WriteString("\n")
			buf.Write(js)
		}
	}
	if cfg.ExtraJS != "" {
		buf.WriteString("\n")
		buf.WriteString(cfg.ExtraJS)
	}
	buf.WriteString("\n</script>\n")

	return template.HTML(buf.String()), nil
}

// CSS returns the combined CSS for the editor (for consumers who want
// to serve it separately rather than inline).
func CSS(includeFullscreen bool) string {
	var buf bytes.Buffer
	if css, err := readEmbedded("css/editor.css"); err == nil {
		buf.Write(css)
	}
	if includeFullscreen {
		if css, err := readEmbedded("css/fullscreen.css"); err == nil {
			buf.WriteString("\n")
			buf.Write(css)
		}
	}
	return buf.String()
}

// JS returns the combined JS for the editor.
func JS(includeFullscreen bool) string {
	var buf bytes.Buffer
	if js, err := readEmbedded("js/editor-core.js"); err == nil {
		buf.Write(js)
	}
	if includeFullscreen {
		if js, err := readEmbedded("js/fullscreen.js"); err == nil {
			buf.WriteString("\n")
			buf.Write(js)
		}
	}
	if js, err := readEmbedded("js/keyboard.js"); err == nil {
		buf.WriteString("\n")
		buf.Write(js)
	}
	return buf.String()
}

// AnnotationsJS returns the standalone annotations JavaScript module source.
// Serve this to your frontend (e.g. at /wiki/annotations.js) to enable
// the go-wiki text annotation API (window.GoWikiAnnotations).
// This is independent of EditorHTML — use it when embedding the editor
// in a React/Vue/Svelte app that manages annotations externally.
func AnnotationsJS() string {
	js, _ := readEmbedded("js/annotations.js")
	return string(js)
}

func readEmbedded(path string) ([]byte, error) {
	return embeddedFS.ReadFile(path)
}
