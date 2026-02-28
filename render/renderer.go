package render

import (
	"strings"

	"github.com/russross/blackfriday/v2"
)

// RendererOptions toggles features without touching code.
type RendererOptions struct {
	AddListClasses       bool   // add classes to <ul>/<ol>/<li>
	AddBlockquoteClasses bool   // style blockquotes
	EnableLightbox       bool   // wrap images in anchors for lightbox
	EnableYouTubeEmbeds  bool   // turn plain YT links into iframes
	EnableTaskListHTML   bool   // turn - [x] into checkboxes
	EnableMermaid        bool   // convert ```mermaid to <div class="mermaid">
	ProtectInlineCSS     bool   // strip one-line CSS outsiders pasted accidentally
	EnableDrawEmbeds     bool   // turn [draw:id] shortcodes into go-draw iframes
	DrawBasePath         string // URL prefix for go-draw (e.g. "/draw"), required when EnableDrawEmbeds is true
}

// ClassConfig allows consumers to override the default CSS classes
// applied by the list and blockquote post-processing filters.
type ClassConfig struct {
	ULClass         string // default: "list-disc pl-2"
	OLClass         string // default: "list-decimal pl-2"
	LIClass         string // default: "mb-2"
	BlockquoteClass string // default: Tailwind blockquote classes
}

// DefaultClassConfig returns Tailwind-compatible classes used by the blog.
func DefaultClassConfig() ClassConfig {
	return ClassConfig{
		ULClass:         "list-disc pl-2",
		OLClass:         "list-decimal pl-2",
		LIClass:         "mb-2",
		BlockquoteClass: `p-4 my-4 border-s-4 border-gray-300 bg-gray-50 dark:border-gray-500 dark:bg-gray-800`,
	}
}

// DefaultOptions returns sane defaults with all features enabled.
func DefaultOptions() RendererOptions {
	return RendererOptions{
		AddListClasses:       true,
		AddBlockquoteClasses: true,
		EnableLightbox:       true,
		EnableYouTubeEmbeds:  true,
		EnableTaskListHTML:   true,
		EnableMermaid:        true,
		ProtectInlineCSS:     true,
		EnableDrawEmbeds:     false, // opt-in; requires DrawBasePath
	}
}

// MinimalOptions returns options with all features disabled —
// only the core markdown rendering with no post-processing.
func MinimalOptions() RendererOptions {
	return RendererOptions{}
}

// Renderer holds the configuration for the markdown rendering pipeline.
type Renderer struct {
	Opt     RendererOptions
	Classes ClassConfig
}

// NewRenderer creates a Renderer with DefaultClassConfig.
func NewRenderer(opt RendererOptions) *Renderer {
	return &Renderer{Opt: opt, Classes: DefaultClassConfig()}
}

// NewRendererWithClasses creates a Renderer with custom CSS classes.
func NewRendererWithClasses(opt RendererOptions, classes ClassConfig) *Renderer {
	return &Renderer{Opt: opt, Classes: classes}
}

// Render runs the full pipeline and returns final HTML.
func (r *Renderer) Render(content string) string {
	html, _ := r.RenderWithDebug(content, false)
	return html
}

// RenderWithDebug returns the final HTML and (optionally) every stage output for inspection.
func (r *Renderer) RenderWithDebug(content string, includeStages bool) (string, map[string]string) {
	stages := map[string]string{}

	stage := func(name, s string) string {
		if includeStages {
			stages[name] = s
		}
		return s
	}

	// --- PRE ---
	s := content
	s = stage("00_raw", s)
	s = normalizeWhitespaceAndBreaks(s)
	s = stage("01_normalized", s)

	if r.Opt.ProtectInlineCSS {
		s = stripStyleSnippets(s)
		s = stage("02_strip_style_snippets", s)
	}

	s = replaceMoreTag(s)
	s = stage("03_more_tag", s)

	s = unwrapListLikeContainers(s)
	s = ensureListSeparation(s)
	s = preprocessLooseMarkdownHTML(s)
	s = normalizeInlinePipeTables(s)
	s = convertFences(s)
	s = stage("04_preprocessed", s)

	// --- MARKDOWN ---
	md := renderMarkdown(s)
	md = stage("05_markdown", md)

	// --- POST ---
	if r.Opt.EnableMermaid {
		md = transformMermaidBlocks(md)
		md = stage("06_mermaid", md)
	}
	if r.Opt.EnableTaskListHTML {
		md = taskListToHTML(md)
		md = stage("07_tasklist", md)
	}
	if r.Opt.EnableYouTubeEmbeds {
		md = embedYouTube(md)
		md = stage("08_youtube", md)
	}
	if r.Opt.EnableDrawEmbeds && r.Opt.DrawBasePath != "" {
		md = embedDraw(md, r.Opt.DrawBasePath)
		md = stage("08b_draw", md)
	}
	if r.Opt.AddListClasses {
		md = addListClasses(md, r.Classes)
		md = stage("09_list_classes", md)
	}
	if r.Opt.AddBlockquoteClasses {
		md = addBlockquoteClasses(md, r.Classes)
		md = stage("10_blockquote_classes", md)
	}
	md = convertInlineEmphasisInHTML(md)
	md = stage("11_inline_emphasis", md)

	if r.Opt.EnableLightbox {
		md = wrapImageGalleries(md)
		md = stage("12_lightbox", md)
	}

	final := md
	return final, stages
}

// ---- Markdown renderer ----

func renderMarkdown(content string) string {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	exts := blackfriday.CommonExtensions |
		blackfriday.AutoHeadingIDs |
		blackfriday.FencedCode |
		blackfriday.Tables |
		blackfriday.Strikethrough
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{})
	out := blackfriday.Run([]byte(content), blackfriday.WithExtensions(exts), blackfriday.WithRenderer(renderer))
	return string(out)
}
