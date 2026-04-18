package render

import (
	"fmt"
	"regexp"
	"strings"
)

// KaTeX/MathML protection: blackfriday's markdown renderer mangles complex
// <span class="katex"> blocks and raw LaTeX delimiters ($$...$$ and $...$).
// We extract them before rendering and restore after.

var (
	// Match <span class="katex">...</span> blocks (including katex-display).
	// These are nested, so we use a simple state machine rather than a regex.
	reKatexStart = regexp.MustCompile(`<span class="katex(?:-display)?">`)

	// Raw LaTeX delimiters — protected so markdown doesn't mangle underscores/backslashes.
	// Display math: $$...$$ (can span multiple lines)
	reMathDisplay = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
	// Inline math: $...$ (no spaces at boundaries, not $50 style)
	reMathInline = regexp.MustCompile(`\$([^\s$][^$]*?[^\s$])\$`)
)

// katexPlaceholders holds extracted KaTeX blocks keyed by placeholder string.
type katexPlaceholders struct {
	blocks []string
}

// protectKaTeX replaces all <span class="katex">...</span> blocks AND raw
// LaTeX delimiters ($$...$$ and $...$) with placeholders that survive
// markdown processing.
func protectKaTeX(s string) (string, *katexPlaceholders) {
	kp := &katexPlaceholders{}

	// First: protect raw LaTeX delimiters ($$...$$ then $...$)
	s = reMathDisplay.ReplaceAllStringFunc(s, func(m string) string {
		ph := fmt.Sprintf("\n\n<!--KATEX_%d-->\n\n", len(kp.blocks))
		kp.blocks = append(kp.blocks, m)
		return ph
	})
	s = reMathInline.ReplaceAllStringFunc(s, func(m string) string {
		ph := fmt.Sprintf("<!--KATEX_%d-->", len(kp.blocks))
		kp.blocks = append(kp.blocks, m)
		return ph
	})

	// Then: protect pre-rendered <span class="katex"> HTML blocks
	for {
		loc := reKatexStart.FindStringIndex(s)
		if loc == nil {
			break
		}
		start := loc[0]

		// Find the matching closing </span> by counting depth
		depth := 0
		pos := start
		end := -1
		for pos < len(s) {
			if strings.HasPrefix(s[pos:], "<span") {
				depth++
				pos += 5
			} else if strings.HasPrefix(s[pos:], "</span>") {
				depth--
				if depth == 0 {
					end = pos + 7 // len("</span>")
					break
				}
				pos += 7
			} else {
				pos++
			}
		}

		if end == -1 {
			break // unclosed tag, stop
		}

		block := s[start:end]
		placeholder := fmt.Sprintf("\n\n<!--KATEX_%d-->\n\n", len(kp.blocks))
		kp.blocks = append(kp.blocks, block)
		s = s[:start] + placeholder + s[end:]
	}

	return s, kp
}

// restoreKaTeX puts the original KaTeX blocks back in place of placeholders.
func restoreKaTeX(s string, kp *katexPlaceholders) string {
	if kp == nil {
		return s
	}
	for i, block := range kp.blocks {
		placeholder := fmt.Sprintf("<!--KATEX_%d-->", i)
		// The placeholder might be wrapped in <p> tags by the markdown renderer
		s = strings.Replace(s, "<p>"+placeholder+"</p>", block, 1)
		s = strings.Replace(s, placeholder, block, 1)
	}
	return s
}
