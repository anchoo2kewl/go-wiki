package render

import (
	"strings"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if !opts.AddListClasses {
		t.Error("Expected AddListClasses to be true")
	}
	if !opts.AddBlockquoteClasses {
		t.Error("Expected AddBlockquoteClasses to be true")
	}
	if !opts.EnableLightbox {
		t.Error("Expected EnableLightbox to be true")
	}
	if !opts.EnableYouTubeEmbeds {
		t.Error("Expected EnableYouTubeEmbeds to be true")
	}
	if !opts.EnableTaskListHTML {
		t.Error("Expected EnableTaskListHTML to be true")
	}
	if !opts.EnableMermaid {
		t.Error("Expected EnableMermaid to be true")
	}
	if !opts.ProtectInlineCSS {
		t.Error("Expected ProtectInlineCSS to be true")
	}
}

func TestMinimalOptions(t *testing.T) {
	opts := MinimalOptions()
	if opts.AddListClasses || opts.AddBlockquoteClasses || opts.EnableLightbox ||
		opts.EnableYouTubeEmbeds || opts.EnableTaskListHTML || opts.EnableMermaid || opts.ProtectInlineCSS {
		t.Error("Expected all MinimalOptions to be false")
	}
}

func TestNewRenderer(t *testing.T) {
	opts := RendererOptions{
		AddListClasses: true,
		EnableLightbox: false,
	}

	r := NewRenderer(opts)

	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}

	if r.Opt.AddListClasses != true {
		t.Error("Expected AddListClasses to be true")
	}

	if r.Opt.EnableLightbox != false {
		t.Error("Expected EnableLightbox to be false")
	}

	// Should use default class config
	if r.Classes.ULClass != "list-disc pl-2" {
		t.Errorf("Expected default ULClass, got: %s", r.Classes.ULClass)
	}
}

func TestNewRendererWithClasses(t *testing.T) {
	opts := DefaultOptions()
	cc := ClassConfig{
		ULClass:         "custom-ul",
		OLClass:         "custom-ol",
		LIClass:         "custom-li",
		BlockquoteClass: "custom-bq",
	}

	r := NewRendererWithClasses(opts, cc)

	if r.Classes.ULClass != "custom-ul" {
		t.Errorf("Expected custom ULClass, got: %s", r.Classes.ULClass)
	}
}

func TestRender(t *testing.T) {
	r := NewRenderer(DefaultOptions())

	t.Run("renders basic markdown", func(t *testing.T) {
		input := "# Hello World\n\nThis is a paragraph."
		output := r.Render(input)

		if !strings.Contains(output, "<h1") {
			t.Error("Expected h1 tag in output")
		}
		if !strings.Contains(output, "Hello World") {
			t.Error("Expected 'Hello World' in output")
		}
		if !strings.Contains(output, "<p>") {
			t.Error("Expected p tag in output")
		}
		if !strings.Contains(output, "This is a paragraph") {
			t.Error("Expected paragraph text in output")
		}
	})

	t.Run("renders bold text", func(t *testing.T) {
		input := "This is **bold** text."
		output := r.Render(input)

		if !strings.Contains(output, "<strong>bold</strong>") {
			t.Error("Expected <strong> tag for bold text")
		}
	})

	t.Run("renders italic text", func(t *testing.T) {
		input := "This is *italic* text."
		output := r.Render(input)

		if !strings.Contains(output, "<em>italic</em>") {
			t.Error("Expected <em> tag for italic text")
		}
	})

	t.Run("renders links", func(t *testing.T) {
		input := "[Link Text](https://example.com)"
		output := r.Render(input)

		if !strings.Contains(output, `<a href="https://example.com"`) {
			t.Error("Expected anchor tag with href")
		}
		if !strings.Contains(output, "Link Text") {
			t.Error("Expected link text in output")
		}
	})

	t.Run("renders code blocks", func(t *testing.T) {
		input := "```go\nfunc main() {}\n```"
		output := r.Render(input)

		if !strings.Contains(output, "<pre>") {
			t.Error("Expected <pre> tag for code block")
		}
		if !strings.Contains(output, "<code") {
			t.Error("Expected <code> tag for code block")
		}
		if !strings.Contains(output, "func main()") {
			t.Error("Expected code content in output")
		}
	})

	t.Run("renders inline code", func(t *testing.T) {
		input := "Use the `fmt.Println` function."
		output := r.Render(input)

		if !strings.Contains(output, "<code>") {
			t.Error("Expected <code> tag for inline code")
		}
		if !strings.Contains(output, "fmt.Println") {
			t.Error("Expected code content in output")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		output := r.Render("")

		if output == "" {
			return
		}
		if len(output) > 50 {
			t.Errorf("Expected minimal output for empty input, got: %s", output)
		}
	})

	t.Run("handles whitespace", func(t *testing.T) {
		input := "   \n\n   \n"
		output := r.Render(input)

		if len(output) > 100 {
			t.Errorf("Expected minimal output for whitespace, got length: %d", len(output))
		}
	})
}

func TestRenderWithDebug(t *testing.T) {
	r := NewRenderer(DefaultOptions())

	t.Run("returns HTML and stages with debug enabled", func(t *testing.T) {
		input := "# Test"
		html, stages := r.RenderWithDebug(input, true)

		if html == "" {
			t.Error("Expected non-empty HTML output")
		}

		if !strings.Contains(html, "<h1") {
			t.Error("Expected h1 tag in HTML output")
		}

		if len(stages) == 0 {
			t.Error("Expected non-empty stages map when debug is enabled")
		}

		if _, exists := stages["00_raw"]; !exists {
			t.Error("Expected '00_raw' stage")
		}
	})

	t.Run("returns HTML without stages when debug disabled", func(t *testing.T) {
		input := "# Test"
		html, stages := r.RenderWithDebug(input, false)

		if html == "" {
			t.Error("Expected non-empty HTML output")
		}

		if len(stages) != 0 {
			t.Error("Expected empty stages map when debug is disabled")
		}
	})
}

func TestRendererOptions(t *testing.T) {
	t.Run("renders with AddListClasses disabled", func(t *testing.T) {
		opts := DefaultOptions()
		opts.AddListClasses = false
		r := NewRenderer(opts)

		input := "- Item 1\n- Item 2"
		output := r.Render(input)

		if !strings.Contains(output, "<ul>") {
			t.Error("Expected <ul> tag")
		}
		if !strings.Contains(output, "<li>") {
			t.Error("Expected <li> tag")
		}
	})

	t.Run("renders blockquotes", func(t *testing.T) {
		r := NewRenderer(DefaultOptions())

		input := "> This is a quote"
		output := r.Render(input)

		if !strings.Contains(output, "This is a quote") {
			t.Error("Expected quote text in output")
		}
		if output == input {
			t.Error("Expected rendered HTML, not raw markdown")
		}
	})

	t.Run("renders ordered lists", func(t *testing.T) {
		r := NewRenderer(DefaultOptions())

		input := "1. First\n2. Second\n3. Third"
		output := r.Render(input)

		if !strings.Contains(output, "First") {
			t.Error("Expected 'First' in output")
		}
		if !strings.Contains(output, "Second") {
			t.Error("Expected 'Second' in output")
		}
		if !strings.Contains(output, "Third") {
			t.Error("Expected 'Third' in output")
		}
	})

	t.Run("renders unordered lists", func(t *testing.T) {
		r := NewRenderer(DefaultOptions())

		input := "- Apple\n- Banana\n- Cherry"
		output := r.Render(input)

		if !strings.Contains(output, "Apple") {
			t.Error("Expected 'Apple' in output")
		}
		if !strings.Contains(output, "Banana") {
			t.Error("Expected 'Banana' in output")
		}
		if !strings.Contains(output, "Cherry") {
			t.Error("Expected 'Cherry' in output")
		}
	})

	t.Run("renders headings at different levels", func(t *testing.T) {
		r := NewRenderer(DefaultOptions())

		tests := []struct {
			input string
			tag   string
		}{
			{"# H1", "<h1"},
			{"## H2", "<h2"},
			{"### H3", "<h3"},
			{"#### H4", "<h4"},
			{"##### H5", "<h5"},
			{"###### H6", "<h6"},
		}

		for _, tt := range tests {
			output := r.Render(tt.input)
			if !strings.Contains(output, tt.tag) {
				t.Errorf("Expected %s tag for input %q", tt.tag, tt.input)
			}
		}
	})
}

func TestMoreTagReplacement(t *testing.T) {
	r := NewRenderer(DefaultOptions())

	t.Run("removes <more--> tag", func(t *testing.T) {
		input := "First paragraph\n\n<more-->\n\nSecond paragraph"
		output := r.Render(input)

		if strings.Contains(output, "<more-->") {
			t.Error("Expected <more--> tag to be removed")
		}

		if !strings.Contains(output, "First paragraph") {
			t.Error("Expected first paragraph in output")
		}
		if !strings.Contains(output, "Second paragraph") {
			t.Error("Expected second paragraph in output")
		}
	})

	t.Run("handles content without more tag", func(t *testing.T) {
		input := "Just one paragraph"
		output := r.Render(input)

		if !strings.Contains(output, "Just one paragraph") {
			t.Error("Expected paragraph text in output")
		}
	})
}

func TestComplexMarkdown(t *testing.T) {
	r := NewRenderer(DefaultOptions())

	t.Run("renders nested lists", func(t *testing.T) {
		input := `- Item 1
  - Nested 1
  - Nested 2
- Item 2`
		output := r.Render(input)

		if !strings.Contains(output, "Item 1") {
			t.Error("Expected 'Item 1' in output")
		}
		if !strings.Contains(output, "Nested 1") {
			t.Error("Expected 'Nested 1' in output")
		}
		if !strings.Contains(output, "Item 2") {
			t.Error("Expected 'Item 2' in output")
		}
	})

	t.Run("renders tables", func(t *testing.T) {
		input := `| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |`
		output := r.Render(input)

		if !strings.Contains(output, "<table>") {
			t.Error("Expected <table> tag")
		}
		if !strings.Contains(output, "Header 1") {
			t.Error("Expected table headers in output")
		}
	})

	t.Run("renders mixed content", func(t *testing.T) {
		input := `# Title

This is **bold** and *italic*.

- List item
- Another item

` + "```" + `
code block
` + "```"
		output := r.Render(input)

		if !strings.Contains(output, "Title") {
			t.Error("Expected 'Title' in output")
		}
		if !strings.Contains(output, "bold") {
			t.Error("Expected 'bold' text in output")
		}
		if !strings.Contains(output, "italic") {
			t.Error("Expected 'italic' text in output")
		}
		if !strings.Contains(output, "List item") {
			t.Error("Expected 'List item' in output")
		}
		if !strings.Contains(output, "code block") {
			t.Error("Expected 'code block' in output")
		}
	})
}

func TestCustomClassConfig(t *testing.T) {
	cc := ClassConfig{
		ULClass:         "my-ul",
		OLClass:         "my-ol",
		LIClass:         "my-li",
		BlockquoteClass: "my-bq",
	}
	r := NewRendererWithClasses(DefaultOptions(), cc)

	t.Run("uses custom UL class", func(t *testing.T) {
		input := "- Item 1\n- Item 2"
		output := r.Render(input)

		if !strings.Contains(output, `class="my-ul"`) {
			t.Errorf("Expected custom UL class, got: %s", output)
		}
		if !strings.Contains(output, `class="my-li"`) {
			t.Errorf("Expected custom LI class, got: %s", output)
		}
	})

	t.Run("uses custom blockquote class", func(t *testing.T) {
		input := "> A quote"
		output := r.Render(input)

		if !strings.Contains(output, `class="my-bq"`) {
			t.Errorf("Expected custom blockquote class, got: %s", output)
		}
	})
}

func TestReferences(t *testing.T) {
	r := NewRenderer(DefaultOptions())

	t.Run("basic citation and definition", func(t *testing.T) {
		input := "See this claim[^1] for details.\n\n[^1]: Knuth, The Art of Computer Programming, 1968"
		output := r.Render(input)

		// Inline citation should be superscript with link
		if !strings.Contains(output, `<sup>`) {
			t.Error("Expected <sup> tag for citation")
		}
		if !strings.Contains(output, `href="#gw-ref-1"`) {
			t.Error("Expected href to #gw-ref-1")
		}
		if !strings.Contains(output, `[1]</a></sup>`) {
			t.Error("Expected [1] as link text in superscript")
		}
		if !strings.Contains(output, `style="color:#3b82f6`) {
			t.Error("Expected blue color style")
		}

		// Reference list should appear
		if !strings.Contains(output, `id="gw-ref-1"`) {
			t.Error("Expected reference anchor id gw-ref-1")
		}
		if !strings.Contains(output, `Knuth, The Art of Computer Programming`) {
			t.Error("Expected reference text in output")
		}
		if !strings.Contains(output, `class="gowiki-references"`) {
			t.Error("Expected gowiki-references section")
		}
		// Back-link
		if !strings.Contains(output, `href="#gw-cite-1"`) {
			t.Error("Expected back-link to gw-cite-1")
		}
	})

	t.Run("multiple references", func(t *testing.T) {
		input := "First[^1] and second[^2].\n\n[^1]: Reference one\n[^2]: Reference two"
		output := r.Render(input)

		if !strings.Contains(output, `[1]</a></sup>`) {
			t.Error("Expected [1] citation")
		}
		if !strings.Contains(output, `[2]</a></sup>`) {
			t.Error("Expected [2] citation")
		}
		if !strings.Contains(output, `id="gw-ref-1"`) {
			t.Error("Expected gw-ref-1 anchor")
		}
		if !strings.Contains(output, `id="gw-ref-2"`) {
			t.Error("Expected gw-ref-2 anchor")
		}
		if !strings.Contains(output, "Reference one") {
			t.Error("Expected 'Reference one'")
		}
		if !strings.Contains(output, "Reference two") {
			t.Error("Expected 'Reference two'")
		}
	})

	t.Run("citation inside inline code is not converted", func(t *testing.T) {
		input := "Use `[^1]` syntax for references.\n\n[^1]: Some ref"
		output := r.Render(input)

		// The inline code should contain literal [^1]
		if !strings.Contains(output, "<code>") {
			t.Error("Expected code tag")
		}
	})

	t.Run("no references does nothing", func(t *testing.T) {
		input := "Just a normal paragraph."
		output := r.Render(input)

		if strings.Contains(output, "gowiki-references") {
			t.Error("Expected no reference section for content without references")
		}
	})

	t.Run("references disabled", func(t *testing.T) {
		opts := DefaultOptions()
		opts.EnableReferences = false
		noRefRenderer := NewRenderer(opts)

		input := "Claim[^1].\n\n[^1]: Source"
		output := noRefRenderer.Render(input)

		if strings.Contains(output, "gowiki-references") {
			t.Error("Expected no reference processing when disabled")
		}
	})

	t.Run("definition removed from body", func(t *testing.T) {
		input := "Text[^1].\n\n[^1]: My source reference"
		output := r.Render(input)

		// The raw definition line should not appear as a paragraph
		if strings.Contains(output, "[^1]: My source reference") {
			t.Error("Expected raw definition line to be removed from body")
		}
	})

	t.Run("markdown links in definitions become clickable", func(t *testing.T) {
		input := "Claim[^1].\n\n[^1]: MSCI — [Full Report](https://example.com/report) (May 2025)"
		output := r.Render(input)

		if !strings.Contains(output, `<a href="https://example.com/report"`) {
			t.Error("Expected markdown link converted to <a> tag in reference definition")
		}
		if !strings.Contains(output, `>Full Report</a>`) {
			t.Error("Expected link text 'Full Report' in anchor tag")
		}
		if strings.Contains(output, `[Full Report](https://example.com/report)`) {
			t.Error("Raw markdown link should not appear in output")
		}
	})

	t.Run("references section has heading", func(t *testing.T) {
		input := "Claim[^1].\n\n[^1]: Source"
		output := r.Render(input)

		if !strings.Contains(output, `id="references"`) {
			t.Error("Expected references heading with id")
		}
		if !strings.Contains(output, ">References<") {
			t.Error("Expected 'References' heading text")
		}
	})
}

func BenchmarkRender(b *testing.B) {
	r := NewRenderer(DefaultOptions())
	input := `# Heading

This is a **paragraph** with *emphasis*.

- Item 1
- Item 2

` + "```go\nfunc main() {}\n```"

	for i := 0; i < b.N; i++ {
		r.Render(input)
	}
}

func BenchmarkRenderSimple(b *testing.B) {
	r := NewRenderer(DefaultOptions())
	input := "This is a simple paragraph."

	for i := 0; i < b.N; i++ {
		r.Render(input)
	}
}

func BenchmarkRenderComplex(b *testing.B) {
	r := NewRenderer(DefaultOptions())
	input := strings.Repeat(`# Heading

This is a **paragraph** with *emphasis* and [a link](https://example.com).

- Item 1
- Item 2
- Item 3

`, 10)

	for i := 0; i < b.N; i++ {
		r.Render(input)
	}
}
