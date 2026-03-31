package render

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Package-level compiled regexps (compiled once at startup).
var (
	// stripStyleSnippets
	reStripPreCodeCSS = regexp.MustCompile(`(?m)^\s*pre\s*code\s*\{[^}]*\}\s*$`)
	reStripTokenCSS   = regexp.MustCompile(`(?m)^\s*\.[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*\s*\{[^}]*\}\s*$`)

	// unwrapListLikeContainers
	reUnwrapListDiv       = regexp.MustCompile(`(?is)<div\b[^>]*>\s*([\-\*\+]\s*|\d+\.\s*)([\s\S]*?)</div>`)
	reUnwrapListP         = regexp.MustCompile(`(?is)<p\b[^>]*>\s*([\-\*\+]\s*|\d+\.\s*)([\s\S]*?)</p>`)
	reMergeConsecutiveList = regexp.MustCompile(`(?m)\n([-\*\+] |\d+\. ).+\n(?:(?:[-\*\+] |\d+\. ).+\n)+`)

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
	rePipeTablePara  = regexp.MustCompile(`(?is)<p>([\s\S]*?\|[\s\S]*?)</p>`)
	rePipeConcat     = regexp.MustCompile(`\|\|`)  // matches "||" — directly concatenated row boundaries
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
	referenceDefRe  = regexp.MustCompile(`(?m)^\[\^(\d+)\]:\s+(.+)$`)
	referenceCiteRe = regexp.MustCompile(`\[\^(\d+)\]`)
	inlineCodeRe    = regexp.MustCompile("`[^`]+`")
	mdLinkRe        = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
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

// preprocessLooseMarkdownHTML converts headings/quotes inside plain HTML containers
// and adds blank lines after block containers so markdown resumes cleanly.
func preprocessLooseMarkdownHTML(content string) string {
	return protectPreBlocks(content, func(content string) string {
		content = reCloseBlock.ReplaceAllString(content, "</$1>\n\n")

		content = reTopLevelH3.ReplaceAllString(content, `<h3>$1</h3>`)
		content = reTopLevelH2.ReplaceAllString(content, `<h2>$1</h2>`)
		content = reTopLevelH1.ReplaceAllString(content, `<h1>$1</h1>`)

		content = processBlockquotes(content)

		content = reParaH3.ReplaceAllString(content, `<h3>$1</h3>`)
		content = reParaH2.ReplaceAllString(content, `<h2>$1</h2>`)
		content = reParaH1.ReplaceAllString(content, `<h1>$1</h1>`)

		content = reParaUL.ReplaceAllString(content, "\n$1$2 $3\n")
		content = reDivUL.ReplaceAllString(content, "\n$1$2 $3\n")

		content = reParaOL.ReplaceAllString(content, "\n$1$2. $3\n")
		content = reDivOL.ReplaceAllString(content, "\n$1$2. $3\n")

		content = reIndent2UL.ReplaceAllString(content, `    $1`)
		content = reIndent2OL.ReplaceAllString(content, `    $1`)

		content = reParaEmphasis.ReplaceAllString(content, "\n$1\n")

		content = reTopLevelHR.ReplaceAllString(content, `<hr/>`)
		content = reParaHR.ReplaceAllString(content, `<hr/>`)

		content = reInnerH3.ReplaceAllString(content, `><h3>$2</h3><`)
		content = reInnerH2.ReplaceAllString(content, `><h2>$2</h2><`)
		content = reInnerH1.ReplaceAllString(content, `><h1>$2</h1><`)

		return content
	})
}

// normalizeInlinePipeTables normalizes inline pipe tables that were collapsed into a single line.
// Handles "| |" (space-separated), "||" (directly concatenated) row boundaries,
// and text)| table start (heading/text merging into first table row).
func normalizeInlinePipeTables(content string) string {
	return protectPreBlocks(content, func(s string) string {
		s = rePipeTablePara.ReplaceAllStringFunc(s, func(p string) string {
			if strings.Count(p, "|") >= 8 || strings.Contains(p, "---") {
				p = strings.ReplaceAll(p, "| |", "|\n|")
				p = rePipeConcat.ReplaceAllString(p, "|\n|")
				return p
			}
			return p
		})
		if strings.Count(s, "| |") >= 2 {
			s = strings.ReplaceAll(s, "| |", "|\n|")
		}
		// Handle collapsed tables (identified by "||" + "---" markers)
		if strings.Count(s, "||") >= 2 && strings.Contains(s, "---") {
			s = rePipeConcat.ReplaceAllString(s, "|\n|")
		}
		// When content has all newlines stripped, restore structural breaks:
		// 1. Before markdown headings (## ) that are glued to previous text
		// 2. Between table rows and non-table text
		if strings.Contains(s, "|---") || strings.Count(s, "|") >= 6 {
			// Restore newlines before markdown headings embedded mid-line
			s = reCollapsedHeading.ReplaceAllString(s, "$1\n\n$2")

			// Fix lines where non-pipe text runs into a table row start
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if idx := findTableStartInLine(line); idx > 0 {
					lines[i] = line[:idx] + "\n\n" + line[idx:]
				}
			}
			s = strings.Join(lines, "\n")
		}
		return s
	})
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

// processReferences converts [^N] inline citations to superscript links and
// [^N]: text definitions into a reference list appended at the end.
//
// Syntax:
//
//	Inline:     [^1]  →  <sup><a href="#gw-ref-1">[1]</a></sup>  (blue, superscript)
//	Definition: [^1]: Knuth, The Art of Computer Programming, 1968
//
// Definitions can appear anywhere; they are collected, removed from the body,
// and rendered as an ordered list at the bottom with back-links.
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

	// Protect <pre> blocks (fenced code already converted by convertFences).
	content = protectPreBlocks(content, func(s string) string {
		// 1. Extract reference definitions.
		defs := map[int]string{}
		var order []int
		s = referenceDefRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := referenceDefRe.FindStringSubmatch(m)
			n := atoiSimple(sub[1])
			if _, exists := defs[n]; !exists {
				order = append(order, n)
			}
			defs[n] = sub[2]
			return "" // remove definition line
		})

		if len(defs) == 0 {
			return s
		}

		// 2. Replace [^N] with superscript links.
		citeCounts := map[string]int{}
		s = referenceCiteRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := referenceCiteRe.FindStringSubmatch(m)
			num := sub[1]
			citeCounts[num]++
			idAttr := ""
			if citeCounts[num] == 1 {
				idAttr = ` id="gw-cite-` + num + `"`
			}
			return `<sup><a href="#gw-ref-` + num + `"` + idAttr +
				` style="color:#3b82f6;text-decoration:none">[` + num + `]</a></sup>`
		})

		// 3. Build reference section HTML.
		sort.Ints(order)
		var sb strings.Builder
		sb.WriteString("\n\n<section class=\"gowiki-references\" style=\"margin-top:2rem;padding-top:1rem;border-top:1px solid #e5e7eb\">\n")
		sb.WriteString("<h4 id=\"references\" style=\"font-size:1.1rem;font-weight:600;margin-bottom:0.5rem\">References</h4>\n")
		sb.WriteString("<ol style=\"list-style-type:decimal;padding-left:1.5rem;font-size:0.9em;line-height:1.6\">\n")
		for _, n := range order {
			ns := itoa(n)
			// Convert markdown links [text](url) → <a href="url">text</a> in definition text.
			defHTML := mdLinkRe.ReplaceAllString(defs[n], `<a href="$2" style="color:#3b82f6" target="_blank" rel="noopener">$1</a>`)
			sb.WriteString(fmt.Sprintf(
				"<li id=\"gw-ref-%s\" style=\"margin-bottom:0.25rem\"><a href=\"#gw-cite-%s\" style=\"color:#3b82f6;text-decoration:none;margin-right:0.25rem\" title=\"Back to text\">↩</a>%s</li>\n",
				ns, ns, defHTML,
			))
		}
		sb.WriteString("</ol>\n</section>\n")
		return s + sb.String()
	})

	// Restore inline code.
	for i, m := range codeStash {
		content = strings.ReplaceAll(content, placeholder("REFCODE", i), m)
	}
	return content
}

// atoiSimple converts a decimal string to int (no error handling — caller
// guarantees digits via regex).
func atoiSimple(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
