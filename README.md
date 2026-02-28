# go-wiki

Embeddable markdown wiki editor for Go web apps — fullscreen split-view, live preview, toolbar shortcuts.

## Features

- **12-stage rendering pipeline** — blackfriday markdown + pre/post processing filters (list classes, blockquote styling, YouTube embeds, mermaid diagrams, task lists, lightbox image galleries)
- **Toolbar** — Bold, Italic, H2, H3, UL, OL, Blockquote, HR, Code, Link
- **Fullscreen split-view** — Left: markdown editor, Right: live preview with 350ms debounce
- **Keyboard shortcuts** — F11 toggle fullscreen, Esc exit, Tab indent, Ctrl+Shift+S/M/L image resize
- **Configurable CSS classes** — Override Tailwind classes for any design system
- **Zero frontend dependencies** — Plain vanilla JS, no npm required
- **Embedded assets** — CSS, JS, and templates via Go's `embed.FS`

## Install

```bash
go get github.com/anchoo2kewl/go-wiki
```

## Quick Start

```go
package main

import (
    "html/template"
    "log"
    "net/http"

    gowiki "github.com/anchoo2kewl/go-wiki"
)

func main() {
    wiki := gowiki.New(
        gowiki.WithPreviewEndpoint("/wiki/preview"),
    )

    // Mount preview endpoint
    http.HandleFunc("/wiki/preview", wiki.PreviewHandler())

    // Render markdown to HTML
    html := wiki.RenderContent("# Hello World")

    // Get editor HTML fragment to embed in a page
    editorHTML, _ := wiki.EditorHTML("# Start writing...")

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Packages

| Package | Description |
|---------|-------------|
| `gowiki` (root) | Top-level convenience API (`Wiki` struct) |
| `render` | Markdown rendering pipeline with configurable options |
| `handler` | HTTP handlers (preview, asset serving) |
| `editor` | Editor config, templates, embedded CSS/JS |

## Rendering Options

```go
// All features enabled (default)
opts := render.DefaultOptions()

// No post-processing, just markdown
opts := render.MinimalOptions()

// Custom CSS classes (e.g., for a different design system)
classes := render.ClassConfig{
    ULClass:         "my-list",
    OLClass:         "my-ordered-list",
    LIClass:         "my-item",
    BlockquoteClass: "my-quote",
}
r := render.NewRendererWithClasses(opts, classes)
```

## License

MIT
