// Standalone example showing go-wiki with net/http.
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	gowiki "github.com/anchoo2kewl/go-wiki"
)

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>go-wiki editor demo</title>
</head>
<body>
  <h1>go-wiki editor demo</h1>
  <form method="POST" action="/save">
    {{.EditorHTML}}
    <button type="submit">Save</button>
  </form>
</body>
</html>`))

func main() {
	wiki := gowiki.New(
		gowiki.WithPreviewEndpoint("/wiki/preview"),
	)

	// Mount preview endpoint
	http.HandleFunc("/wiki/preview", wiki.PreviewHandler())

	// Serve editor assets
	http.Handle("/wiki/assets/", http.StripPrefix("/wiki/assets/", wiki.AssetHandler()))

	// Editor page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		editorHTML, err := wiki.EditorHTML("# Hello\n\nStart writing here...")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		pageTmpl.Execute(w, map[string]template.HTML{
			"EditorHTML": editorHTML,
		})
	})

	// Save handler (just echoes the rendered content)
	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		content := r.FormValue("content")
		html := wiki.RenderContent(content)
		fmt.Fprintf(w, "<h1>Rendered Output</h1>\n%s\n<br><a href='/'>Back to editor</a>", html)
	})

	log.Println("go-wiki demo running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
