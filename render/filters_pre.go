package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/russross/blackfriday/v2"
)

// Package-level compiled regexps (compiled once at startup).
var (
	// stripStyleSnippets
	reStripPreCodeCSS = regexp.MustCompile(`(?m)^\s*pre\s*code\s*\{[^}]*\}\s*$`)
	reStripTokenCSS   = regexp.MustCompile(`(?m)^\s*\.[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*\s*\{[^}]*\}\s*$`)

	// unwrapListLikeContainers
	reUnwrapListDiv        = regexp.MustCompile(`(?is)<div\b[^>]*>\s*([\-\*\+]\s*|\d+\.\s*)([\s\S]*?)</div>`)
	reUnwrapListP          = regexp.MustCompile(`(?is)<p\b[^>]*>\s*([\-\*\+]\s*|\d+\.\s*)([\s\S]*?)</p>`)
	reMergeConsecutiveList = regexp.MustCompile(`(?m)\n([-\*\+] |\d+\. ).+\n(?:(?:[-\*\+] |\d+\. ).+\n)+`)

	// escapeSpacedLinkParens: "]" then whitespace then "(".
	reSpacedLinkParen = regexp.MustCompile(`\][ \t]+\(`)

	// normalizeListContinuationIndent: a top-level list item marker.
	reListItemStart = regexp.MustCompile(`^(?:[-*+]|\d+[.)])[ \t]`)

	// ensureListSeparation
	reListSeparation = regexp.MustCompile(`(?m)([^\n])\n([ \t]*)([-*+]|\d+\.)\s+`)

	// preprocessLooseMarkdownHTML
	reCloseBlock   = regexp.MustCompile(`(?is)</(div|figure|section|table|blockquote|p)>\s*`)
	reTopLevelH3   = regexp.MustCompile(`(?m)^[ \t]*###[ \t]+(.+)$`)
	reTopLevelH2   = regexp.MustCompile(`(?m)^[ \t]*##[ \t]+(.+)$`)
	reTopLevelH1   = regexp.MustCompile(`(?m)^[ \t]*#[ \t]+(.+)$`)
	reParaH3       = regexp.MustCompile(`(?is)<p[^>]*>\s*###\s+(.+?)\s*</p>`)
	reParaH2       = regexp.MustCompile(`(?is)<p[^>]*>\s*##\s+(.+?)\s*</p>`)
	reParaH1       = regexp.MustCompile(`(?is)<p[^>]*>\s*#\s+(.+?)\s*</p>`)
	reParaUL       = regexp.MustCompile(`(?is)<p[^>]*>\s*([ \t]*)([-*+])\s+(.+?)\s*</p>`)
	reDivUL        = regexp.MustCompile(`(?is)<div[^>]*>\s*([ \t]*)([-*+])\s+(.+?)\s*</div>`)
	reParaOL       = regexp.MustCompile(`(?is)<p[^>]*>\s*([ \t]*)(\d+)\.\s+(.+?)\s*</p>`)
	reDivOL        = regexp.MustCompile(`(?is)<div[^>]*>\s*([ \t]*)(\d+)\.\s+(.+?)\s*</div>`)
	reIndent2UL    = regexp.MustCompile(`(?m)^[ ]{2}([-*+]\s+)`)
	reIndent2OL    = regexp.MustCompile(`(?m)^[ ]{2}(\d+\.\s+)`)
	reParaEmphasis = regexp.MustCompile(`(?is)<p[^>]*>\s*([^<>]*?(\*\*.+?\*\*|__.+?__|\*[^*]+?\*|_[^_]+?_)\s*[^<>]*?)\s*</p>`)
	reTopLevelHR   = regexp.MustCompile(`(?m)^[ \t]*---[ \t]*$`)
	reParaHR       = regexp.MustCompile(`(?is)<p>\s*---\s*</p>`)
	reInnerH3      = regexp.MustCompile(`(?is)>(\s*###\s+)(.+?)\s*<`)
	reInnerH2      = regexp.MustCompile(`(?is)>(\s*##\s+)(.+?)\s*<`)
	reInnerH1      = regexp.MustCompile(`(?is)>(\s*#\s+)(.+?)\s*<`)

	// normalizeInlinePipeTables
	rePipeTablePara = regexp.MustCompile(`(?is)<p>([\s\S]*?\|[\s\S]*?)</p>`)
	rePipeConcat    = regexp.MustCompile(`\|\|`) // matches "||" — directly concatenated row boundaries
	// A delimiter cell: | --- |, |:--|, |---:| (GFM needs at least one dash).
	rePipeDelimCell = regexp.MustCompile(`\|[ \t]*:?-{3,}:?[ \t]*\|`)
	// A whole delimiter row on its own: | --- | :-: | ---: |
	rePipeDelimRow = regexp.MustCompile(`^\|?(?:[ \t]*:?-+:?[ \t]*\|)+(?:[ \t]*:?-+:?[ \t]*)?$`)
	// Detects markdown headings glued to preceding text when newlines were stripped.
	// e.g. "end of row.## Next Section" → "end of row.\n\n## Next Section"
	reCollapsedHeading = regexp.MustCompile(`([^\n#])(#{1,6}\s+)`)
	// Detects text (non-pipe char) running directly into a pipe table row start.
	// e.g. "heading text)| Header | Value |" → split before the first pipe.
	rePipeTableStart = regexp.MustCompile(`([^\|\n\s])\|(\s*[^\|\-\n][^|]*\|)`) // text)| word |

	// convertFences — opening/closing ``` must be at the start of a line
	// (with up to 3 spaces of indentation per CommonMark spec).
	reCodeFence = regexp.MustCompile("(?m)^[ ]{0,3}```([a-zA-Z0-9_-]*)[ \\t]*\\n([\\s\\S]*?)^[ ]{0,3}```[ \\t]*$")

	// cleanStyleHeader
	reCleanStylePreCode = regexp.MustCompile(`^pre\s+code\s*\{[^}]*\}\s*$`)

	// references
	// Labels follow GFM: letters, digits, "-" and "_" ([^1], [^usc102], [^epc-54]).
	referenceDefRe  = regexp.MustCompile(`(?m)^[ ]{0,3}\[\^([A-Za-z0-9_-]+)\]:[ \t]+(.+)$`)
	referenceCiteRe = regexp.MustCompile(`\[\^([A-Za-z0-9_-]+)\]`)
	inlineCodeRe    = regexp.MustCompile("`[^`]+`")
	referenceLinkRe = regexp.MustCompile(`(<a href="[^"]*")`)
)

// normalizeWhitespaceAndBreaks converts NBSP, line breaks, <br> tags to \n.
func normalizeWhitespaceAndBreaks(content string) string {
	content = strings.NewReplacer(
		"\u00A0", " ",
		"\u2002", " ",
		"\u2003", " ",
		"\u2007", " ",
		"\u202F", " ",
		"&nbsp;", " ",
		"&#160;", " ",
		"\r\n", "\n",
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
	).Replace(content)
	return content
}

// stripStyleSnippets hides single-line CSS rules accidentally pasted (outside code).
func stripStyleSnippets(content string) string {
	return protectPreBlocks(content, func(s string) string {
		s = reStripPreCodeCSS.ReplaceAllString(s, "")
		s = reStripTokenCSS.ReplaceAllString(s, "")
		return s
	})
}

// replaceMoreTag removes <more--> marker (first occurrence, with or without space).
func replaceMoreTag(content string) string {
	markers := []string{"<more-->", "<more -->", "&lt;more--&gt;", "&lt;more --&gt;"}
	for _, mk := range markers {
		if idx := strings.Index(content, mk); idx != -1 {
			return content[:idx] + content[idx+len(mk):]
		}
	}
	return content
}

// unwrapListLikeContainers converts <div>- Item</div> and <p>- Item</p> to markdown list format.
func unwrapListLikeContainers(content string) string {
	replaceFunc := func(matches []string) string {
		if len(matches) < 3 {
			return matches[0]
		}
		marker := strings.TrimSpace(matches[1])
		text := strings.TrimSpace(matches[2])
		if marker != "" && !strings.HasSuffix(marker, " ") {
			marker += " "
		}
		return "\n" + marker + text + "\n"
	}

	s := reUnwrapListDiv.ReplaceAllStringFunc(content, func(m string) string {
		sub := reUnwrapListDiv.FindStringSubmatch(m)
		return replaceFunc(sub)
	})

	s = reUnwrapListP.ReplaceAllStringFunc(s, func(m string) string {
		sub := reUnwrapListP.FindStringSubmatch(m)
		return replaceFunc(sub)
	})

	s = reMergeConsecutiveList.ReplaceAllStringFunc(s, func(block string) string {
		return strings.TrimRight(block, "\n") + "\n\n"
	})
	return s
}

// ensureListSeparation adds a blank line before any list that follows text.
func ensureListSeparation(content string) string {
	return protectPreBlocks(content, func(s string) string {
		return reListSeparation.ReplaceAllString(s, "$1\n\n$2$3 ")
	})
}

// normalizeListContinuationIndent re-indents list-item continuation content
// written with the CommonMark indent (2 spaces after "- ", 3 after "1. ") to
// the 4 spaces blackfriday needs. Without it a paragraph or table that
// follows a blank line inside a list item falls out of the list, and a table
// there renders as plain text.
func normalizeListContinuationIndent(content string) string {
	return protectPreBlocks(content, func(s string) string {
		lines := strings.Split(s, "\n")
		inList := false
		for i, line := range lines {
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			switch {
			case trimmed == "", strings.HasPrefix(trimmed, "\t"):
				continue
			case indent == 0:
				inList = reListItemStart.MatchString(line)
			case inList && indent < 4:
				lines[i] = "    " + trimmed
			}
		}
		return strings.Join(lines, "\n")
	})
}

// escapeSpacedLinkParens stops "[text] (aside)" from becoming a link.
// blackfriday accepts whitespace between "]" and "(", so "Gary [surname] (CTO)"
// rendered "surname" as a link to "CTO" and dropped "(CTO)". CommonMark and GFM
// only form a link when "(" follows "]" directly; escaping the "(" restores
// that. Inline code and <pre> blocks are left untouched.
func escapeSpacedLinkParens(content string) string {
	if !reSpacedLinkParen.MatchString(content) {
		return content
	}
	return protectPreBlocks(content, func(s string) string {
		var codeStash []string
		s = inlineCodeRe.ReplaceAllStringFunc(s, func(m string) string {
			codeStash = append(codeStash, m)
			return placeholder("LINKCODE", len(codeStash)-1)
		})
		s = reSpacedLinkParen.ReplaceAllStringFunc(s, func(m string) string {
			return m[:len(m)-1] + `\(`
		})
		for i, m := range codeStash {
			s = strings.ReplaceAll(s, placeholder("LINKCODE", i), m)
		}
		return s
	})
}

// preprocessLooseMarkdownHTML converts headings/quotes inside plain HTML containers
// and adds blank lines after block containers so markdown resumes cleanly.
func preprocessLooseMarkdownHTML(content string) string {
	return protectPreBlocks(content, func(content string) string {
		content = reCloseBlock.ReplaceAllString(content, "</$1>\n\n")

		// NOTE: Do NOT convert top-level markdown headings (## X) to <h2>X</h2>.
		// Blackfriday handles ## natively and the HTML tags break its table detection.
		// Only convert headings trapped inside <p> tags (lines 156-158 below).

		content = processBlockquotes(content)

		// NOTE: Do NOT convert headings inside <p> tags either — blackfriday handles
		// ## natively, and converting them to <h2> inside <p> creates malformed HTML
		// that nests tables inside headings.

		content = reParaUL.ReplaceAllString(content, "\n$1$2 $3\n")
		content = reDivUL.ReplaceAllString(content, "\n$1$2 $3\n")

		content = reParaOL.ReplaceAllString(content, "\n$1$2. $3\n")
		content = reDivOL.ReplaceAllString(content, "\n$1$2. $3\n")

		content = reIndent2UL.ReplaceAllString(content, `    $1`)
		content = reIndent2OL.ReplaceAllString(content, `    $1`)

		content = reParaEmphasis.ReplaceAllString(content, "\n$1\n")

		content = reTopLevelHR.ReplaceAllString(content, `<hr/>`)
		content = reParaHR.ReplaceAllString(content, `<hr/>`)

		// NOTE: reInnerH1/H2/H3 disabled — same reason as above:
		// blackfriday handles ## natively; converting inside HTML tags
		// creates malformed nesting (tables inside headings).

		return content
	})
}

// normalizeInlinePipeTables repairs pipe tables whose rows were collapsed onto
// a single line (e.g. by Yjs/CRDT sync or LLM output that stripped newlines).
// Handles "| |" (space-separated) and "||" (directly concatenated) row
// boundaries, headings glued to a table, and text)| table starts.
//
// Only lines that are actually collapsed are touched: a line must contain a
// delimiter cell (| --- |) alongside other content. Well-formed GFM tables —
// one row per line — are left alone, so cells such as "#", "`x`" or an empty
// first header cell are never mistaken for headings or row boundaries.
func normalizeInlinePipeTables(content string) string {
	return protectPreBlocks(content, func(s string) string {
		s = rePipeTablePara.ReplaceAllStringFunc(s, func(p string) string {
			if !isCollapsedTableLine(p) {
				return p
			}
			return splitCollapsedRows(p)
		})

		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if !isCollapsedTableLine(line) {
				continue
			}
			line = splitCollapsedRows(line)
			// Restore newlines before markdown headings glued to a row.
			line = reCollapsedHeading.ReplaceAllString(line, "$1\n\n$2")
			// Split text that runs directly into a table row start.
			parts := strings.Split(line, "\n")
			for j, part := range parts {
				if idx := findTableStartInLine(part); idx > 0 {
					parts[j] = part[:idx] + "\n\n" + part[idx:]
				}
			}
			lines[i] = strings.Join(parts, "\n")
		}
		return strings.Join(lines, "\n")
	})
}

// splitCollapsedRows puts each row of a collapsed table back on its own line.
func splitCollapsedRows(s string) string {
	s = strings.ReplaceAll(s, "| |", "|\n|")
	return rePipeConcat.ReplaceAllString(s, "|\n|")
}

// isCollapsedTableLine reports whether a single line holds more than one
// table row: it has a delimiter cell (| --- |, |:---:|) and is not itself
// just a delimiter row.
func isCollapsedTableLine(line string) bool {
	if !rePipeDelimCell.MatchString(line) {
		return false
	}
	return !rePipeDelimRow.MatchString(strings.TrimSpace(line))
}

// findTableStartInLine finds the position where a pipe table row starts within a line
// that begins with non-pipe text. Returns the index of the first pipe, or -1 if not found.
// e.g. "Heading)| Header | Value |" → index of the first |
func findTableStartInLine(line string) int {
	// Must have at least 3 pipes to look like a table row: | h1 | h2 |
	if strings.Count(line, "|") < 3 {
		return -1
	}
	// Line must NOT start with | (already a table row)
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) == 0 || trimmed[0] == '|' {
		return -1
	}
	// Find the first | that starts a table-like pattern: | word | word |
	for i := 1; i < len(line)-1; i++ {
		if line[i] == '|' && line[i-1] != '|' && line[i-1] != '\\' {
			// Check if from this | onward, we have a valid table row pattern
			rest := line[i:]
			pipeCount := strings.Count(rest, "|")
			if pipeCount >= 3 && !strings.HasPrefix(rest, "|---") {
				return i
			}
		}
	}
	return -1
}

// convertFences converts ```lang fences to <pre><code class="language-...">...</code></pre>.
func convertFences(s string) string {
	return reCodeFence.ReplaceAllStringFunc(s, func(m string) string {
		sm := reCodeFence.FindStringSubmatch(m)
		if len(sm) < 3 {
			return m
		}
		lang := strings.TrimSpace(sm[1])
		code := cleanStyleHeader(sm[2])
		return fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, lang, escapeCode(code))
	})
}

func escapeCode(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func cleanStyleHeader(code string) string {
	lines := strings.Split(code, "\n")
	if len(lines) == 0 {
		return code
	}
	first := strings.TrimSpace(lines[0])
	if reCleanStylePreCode.MatchString(first) {
		lines = lines[1:]
	}
	return strings.Join(lines, "\n")
}

// processBlockquotes merges multiple > lines into a single blockquote.
func processBlockquotes(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	var in bool
	var buf []string

	flush := func() {
		if len(buf) > 0 {
			out = append(out, "<blockquote><p>"+strings.Join(buf, " ")+"</p></blockquote>")
			buf = nil
		}
	}

	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, ">") || strings.HasPrefix(t, "&gt;") {
			var text string
			if strings.HasPrefix(t, "&gt;") {
				text = strings.TrimSpace(t[4:])
			} else {
				text = strings.TrimSpace(t[1:])
			}
			in = true
			buf = append(buf, text)
			continue
		}
		if in {
			flush()
			in = false
		}
		out = append(out, ln)
	}
	if in {
		flush()
	}
	return strings.Join(out, "\n")
}

// processReferences converts [^label] inline citations to superscript links and
// [^label]: text definitions into a reference list appended at the end.
//
// Syntax (labels are letters, digits, "-" or "_", as in GFM):
//
//	Inline:     [^usc102]  →  <sup><a href="#gw-ref-usc102">[1]</a></sup>
//	Definition: [^usc102]: 35 U.S.C. 102
//
// References are numbered in order of first citation, like GFM footnotes, so
// the displayed number never depends on the label. Definitions can appear
// anywhere; they are collected, removed from the body, and rendered as an
// ordered list at the bottom with back-links. Definitions that are never cited
// are listed after the cited ones; citations with no definition stay literal.
func processReferences(content string) string {
	// Quick bail-out: if there are no [^ markers at all, skip the work.
	if !strings.Contains(content, "[^") {
		return content
	}

	// Stash inline backtick code so [^N] inside `code` is not converted.
	var codeStash []string
	content = inlineCodeRe.ReplaceAllStringFunc(content, func(m string) string {
		codeStash = append(codeStash, m)
		return placeholder("REFCODE", len(codeStash)-1)
	})
	restoreCode := func(s string) string {
		for i, m := range codeStash {
			s = strings.ReplaceAll(s, placeholder("REFCODE", i), m)
		}
		return s
	}

	// Protect <pre> blocks (fenced code already converted by convertFences).
	content = protectPreBlocks(content, func(s string) string {
		// 1. Extract reference definitions.
		defs := map[string]string{}
		var defOrder []string
		s = referenceDefRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := referenceDefRe.FindStringSubmatch(m)
			label := sub[1]
			if _, exists := defs[label]; !exists {
				defOrder = append(defOrder, label)
			}
			defs[label] = sub[2]
			return "" // remove definition line
		})

		if len(defs) == 0 {
			return s
		}

		// 2. Replace [^label] with superscript links, numbering labels in
		// order of first citation.
		numbers := map[string]int{}
		var order []string
		s = referenceCiteRe.ReplaceAllStringFunc(s, func(m string) string {
			label := referenceCiteRe.FindStringSubmatch(m)[1]
			if _, ok := defs[label]; !ok {
				return m // no definition: leave the text as written
			}
			idAttr := ""
			if _, seen := numbers[label]; !seen {
				order = append(order, label)
				numbers[label] = len(order)
				idAttr = ` id="gw-cite-` + label + `"`
			}
			return `<sup><a href="#gw-ref-` + label + `"` + idAttr +
				` style="color:#3b82f6;text-decoration:none">[` + itoa(numbers[label]) + `]</a></sup>`
		})
		for _, label := range defOrder {
			if _, cited := numbers[label]; !cited {
				order = append(order, label)
			}
		}

		// 3. Build reference section HTML.
		var sb strings.Builder
		sb.WriteString("\n\n<section class=\"gowiki-references\" style=\"margin-top:2rem;padding-top:1rem;border-top:1px solid #e5e7eb\">\n")
		sb.WriteString("<h4 id=\"references\" style=\"font-size:1.1rem;font-weight:600;margin-bottom:0.5rem\">References</h4>\n")
		sb.WriteString("<ol style=\"list-style-type:decimal;padding-left:1.5rem;font-size:0.9em;line-height:1.6\">\n")
		for _, label := range order {
			back := ""
			if _, cited := numbers[label]; cited {
				back = fmt.Sprintf(
					"<a href=\"#gw-cite-%s\" style=\"color:#3b82f6;text-decoration:none;margin-right:0.25rem\" title=\"Back to text\">↩</a>",
					label,
				)
			}
			sb.WriteString(fmt.Sprintf(
				"<li id=\"gw-ref-%s\" style=\"margin-bottom:0.25rem\">%s%s</li>\n",
				label, back, renderReferenceText(restoreCode(defs[label])),
			))
		}
		sb.WriteString("</ol>\n</section>\n")
		return s + sb.String()
	})

	return restoreCode(content)
}

// renderReferenceText renders a definition's inline markdown (links, bare
// URLs, `code`, emphasis). The list sits inside a raw HTML block, which the
// markdown pass does not look into, so it has to be rendered here.
func renderReferenceText(text string) string {
	exts := blackfriday.CommonExtensions | blackfriday.Strikethrough
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.HrefTargetBlank | blackfriday.NoopenerLinks,
	})
	out := string(blackfriday.Run([]byte(text), blackfriday.WithExtensions(exts), blackfriday.WithRenderer(renderer)))
	out = strings.TrimSpace(out)
	out = strings.TrimPrefix(out, "<p>")
	out = strings.TrimSuffix(out, "</p>")
	return referenceLinkRe.ReplaceAllString(out, `$1 style="color:#3b82f6"`)
}
