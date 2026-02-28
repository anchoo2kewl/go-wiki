package render

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-compiled regexes (compiled once at startup)
var (
	listULRe = regexp.MustCompile(`(?i)<ul\b([^>]*)>`)
	listOLRe = regexp.MustCompile(`(?i)<ol\b([^>]*)>`)
	listLIRe = regexp.MustCompile(`(?i)<li\b([^>]*)>`)

	blockquoteRe = regexp.MustCompile(`(?i)<blockquote\b([^>]*)>`)

	taskListRe = regexp.MustCompile(`(?is)<li([^>]*)>\s*\[([ xX])\]\s*(.*?)</li>`)

	youtubeEmbedRe = regexp.MustCompile(`(?is)<p>\s*<a[^>]+href="(https?://(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([^"&<>\s]+)[^"]*)"[^>]*>[^<]*</a>\s*</p>`)

	mermaidBlockRe = regexp.MustCompile(`(?is)<pre><code class="language-mermaid">([\s\S]*?)</code></pre>`)

	emphasisCodeRe       = regexp.MustCompile("(?is)(<pre[\\s\\S]*?</pre>|<code[\\s\\S]*?</code>)")
	emphasisTextNodeRe   = regexp.MustCompile(">([^<]+)<")
	emphasisBoldStarRe   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emphasisBoldUnderRe  = regexp.MustCompile(`__([^_]+)__`)
	emphasisItalicStarRe = regexp.MustCompile(`(^|[^*])\*([^*]+)\*([^*]|$)`)
	emphasisItalicUnderRe = regexp.MustCompile(`(^|[^_])_([^_]+)_([^_]|$)`)

	galleryContainerRe       = regexp.MustCompile(`(?is)(<div[^>]*class="[^"]*image-gallery[^"]*"[^>]*>)([\s\S]*?)(</div>)`)
	galleryImgRe             = regexp.MustCompile(`(?is)<img([^>]*?)\s+src="([^"]+)"([^>]*)>`)
	galleryAltRe             = regexp.MustCompile(`(?i)alt="([^"]*)"`)
	galleryExistingLightboxRe = regexp.MustCompile(`(?is)<a[^>]*data-lightbox[^>]*>\s*<img[^>]*>\s*</a>`)
	galleryFigureRe          = regexp.MustCompile(`(?is)(<figure[^>]*>)([\s\S]*?)(</figure>)`)
	galleryCaptionRe         = regexp.MustCompile(`(?is)<figcaption[^>]*>([\s\S]*?)</figcaption>`)
	galleryHTMLStripRe       = regexp.MustCompile(`<[^>]*>`)
	galleryImgParaRe         = regexp.MustCompile(`(?is)<p>\s*(<img[^>]+>)\s*</p>`)
	galleryParaLightboxRe    = regexp.MustCompile(`(?is)<p>\s*<a[^>]*data-lightbox[^>]*>\s*<img[^>]*>\s*</a>\s*</p>`)
)

// addListClasses adds CSS classes to UL/OL/LI using the provided ClassConfig.
func addListClasses(content string, cc ClassConfig) string {
	withClass := func(tag, classes, attrs string) string {
		if strings.Contains(strings.ToLower(attrs), "class=") {
			return "<" + tag + attrs + ">"
		}
		return "<" + tag + ` class="` + classes + `"` + attrs + ">"
	}

	content = listULRe.ReplaceAllStringFunc(content, func(m string) string {
		attrs := listULRe.FindStringSubmatch(m)[1]
		return withClass("ul", cc.ULClass, attrs)
	})
	content = listOLRe.ReplaceAllStringFunc(content, func(m string) string {
		attrs := listOLRe.FindStringSubmatch(m)[1]
		return withClass("ol", cc.OLClass, attrs)
	})
	content = listLIRe.ReplaceAllStringFunc(content, func(m string) string {
		attrs := listLIRe.FindStringSubmatch(m)[1]
		return withClass("li", cc.LIClass, attrs)
	})
	return content
}

// addBlockquoteClasses adds CSS classes to blockquote elements.
func addBlockquoteClasses(content string, cc ClassConfig) string {
	return blockquoteRe.ReplaceAllStringFunc(content, func(m string) string {
		attrs := blockquoteRe.FindStringSubmatch(m)[1]
		if strings.Contains(strings.ToLower(attrs), "class=") {
			return "<blockquote" + attrs + ">"
		}
		return `<blockquote class="` + cc.BlockquoteClass + `"` + attrs + ">"
	})
}

// taskListToHTML converts "- [x] Task" into disabled checkboxes.
func taskListToHTML(html string) string {
	return taskListRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := taskListRe.FindStringSubmatch(m)
		attrs := sub[1]
		checked := strings.EqualFold(sub[2], "x")
		text := sub[3]
		chk := `<input type="checkbox" disabled` + ternary(checked, " checked", "") + ` class="mr-2 align-middle">`
		if strings.Contains(strings.ToLower(attrs), "class=") {
			return `<li` + attrs + `>` + chk + text + `</li>`
		}
		return `<li class="task-item"` + attrs + `>` + chk + text + `</li>`
	})
}

// embedYouTube replaces paragraph-wrapped YouTube links with responsive iframes.
func embedYouTube(html string) string {
	return youtubeEmbedRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := youtubeEmbedRe.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		id := sub[2]
		iframe := `<div class="aspect-video w-full max-w-3xl"><iframe src="https://www.youtube.com/embed/` + id + `" title="YouTube video" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen style="width:100%;height:100%"></iframe></div>`
		return iframe
	})
}

// transformMermaidBlocks converts Prism-style mermaid code blocks to <div class="mermaid">.
func transformMermaidBlocks(html string) string {
	return mermaidBlockRe.ReplaceAllString(html, `<div class="mermaid">$1</div>`)
}

// convertInlineEmphasisInHTML converts emphasis markers inside already-rendered HTML text nodes.
func convertInlineEmphasisInHTML(html string) string {
	var stash []string
	html = emphasisCodeRe.ReplaceAllStringFunc(html, func(m string) string {
		stash = append(stash, m)
		return placeholder("EMPH", len(stash)-1)
	})

	html = emphasisTextNodeRe.ReplaceAllStringFunc(html, func(seg string) string {
		inner := seg[1 : len(seg)-1]
		if strings.Contains(inner, "[[[") && strings.Contains(inner, "]]]") {
			return seg
		}
		inner = emphasisBoldStarRe.ReplaceAllString(inner, "<strong>$1</strong>")
		inner = emphasisBoldUnderRe.ReplaceAllString(inner, "<strong>$1</strong>")
		inner = emphasisItalicStarRe.ReplaceAllString(inner, "$1<em>$2</em>$3")
		inner = emphasisItalicUnderRe.ReplaceAllString(inner, "$1<em>$2</em>$3")
		return ">" + inner + "<"
	})

	for i, m := range stash {
		html = strings.ReplaceAll(html, placeholder("EMPH", i), m)
	}
	return html
}

// wrapImageGalleries wraps bare <img> tags with lightbox links.
func wrapImageGalleries(html string) string {
	transform := func(content string) string {
		if galleryExistingLightboxRe.MatchString(content) {
			return content
		}

		return galleryImgRe.ReplaceAllStringFunc(content, func(imgTag string) string {
			m := galleryImgRe.FindStringSubmatch(imgTag)
			if len(m) != 4 {
				return imgTag
			}
			preAttrs := m[1]
			src := m[2]
			postAttrs := m[3]
			attrs := preAttrs + " " + postAttrs
			alt := ""
			if am := galleryAltRe.FindStringSubmatch(attrs); len(am) == 2 {
				alt = am[1]
			}
			return fmt.Sprintf(
				`<a href="%s" data-lightbox="article-images" rel="lightbox[article-images]" data-title="%s"><img%s src="%s"%s></a>`,
				src, htmlEscapeAttr(alt), preAttrs, src, postAttrs,
			)
		})
	}

	// 7a) <figure> blocks
	html = galleryFigureRe.ReplaceAllStringFunc(html, func(block string) string {
		if galleryExistingLightboxRe.MatchString(block) {
			return block
		}
		parts := galleryFigureRe.FindStringSubmatch(block)
		if len(parts) != 4 {
			return block
		}
		openTag, inner, closeTag := parts[1], parts[2], parts[3]

		caption := ""
		if cm := galleryCaptionRe.FindStringSubmatch(inner); len(cm) == 2 {
			caption = galleryHTMLStripRe.ReplaceAllString(cm[1], "")
			caption = strings.TrimSpace(caption)
		}

		inner = galleryImgRe.ReplaceAllStringFunc(inner, func(imgTag string) string {
			m := galleryImgRe.FindStringSubmatch(imgTag)
			if len(m) != 4 {
				return imgTag
			}
			preAttrs := m[1]
			src := m[2]
			postAttrs := m[3]
			title := caption
			if title == "" {
				attrs := preAttrs + " " + postAttrs
				if am := galleryAltRe.FindStringSubmatch(attrs); len(am) == 2 {
					title = am[1]
				}
			}
			return fmt.Sprintf(
				`<a href="%s" data-lightbox="article-images" rel="lightbox[article-images]" data-title="%s"><img%s src="%s"%s></a>`,
				src, htmlEscapeAttr(title), preAttrs, src, postAttrs,
			)
		})

		return openTag + inner + closeTag
	})

	// 7b) Inside image-gallery containers
	html = galleryContainerRe.ReplaceAllStringFunc(html, func(block string) string {
		parts := galleryContainerRe.FindStringSubmatch(block)
		if len(parts) != 4 {
			return block
		}
		return parts[1] + transform(parts[2]) + parts[3]
	})

	// 7c) Standalone <p><img ...></p>
	html = galleryImgParaRe.ReplaceAllStringFunc(html, func(m string) string {
		if galleryParaLightboxRe.MatchString(m) {
			return m
		}

		im := galleryImgRe.FindStringSubmatch(m)
		if len(im) != 4 {
			return m
		}
		src := im[2]
		preAttrs := im[1]
		postAttrs := im[3]
		attrs := preAttrs + " " + postAttrs
		alt := ""
		if am := galleryAltRe.FindStringSubmatch(attrs); len(am) == 2 {
			alt = am[1]
		}
		return fmt.Sprintf(
			`<p><a href="%s" data-lightbox="article-images" rel="lightbox[article-images]" data-title="%s"><img%s src="%s"%s></a></p>`,
			src, htmlEscapeAttr(alt), preAttrs, src, postAttrs,
		)
	})

	return html
}
