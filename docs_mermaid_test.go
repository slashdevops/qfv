package qfv

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// GitHub decodes HTML entities before it hands a diagram to Mermaid, so a
// label written with &quot; reaches Mermaid as a quote that ends the label:
// docs/dialects.md's checking diagram rendered as a parse error on GitHub,
// and as a diagram everywhere else. Mermaid's own entity codes (#quot;,
// #lt;, #gt;, #amp;) are left alone by GitHub and decoded by Mermaid.
func TestNoDiagramUsesAnHTMLEntity(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.md")
	if err != nil {
		t.Fatal(err)
	}

	docs, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}

	block := regexp.MustCompile("(?s)```mermaid\n(.*?)```")
	entity := regexp.MustCompile(`&(quot|lt|gt|amp|#[0-9]+);`)
	diagrams := 0

	for _, f := range append(files, docs...) {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		for _, m := range block.FindAllStringSubmatch(string(raw), -1) {
			diagrams++

			if e := entity.FindString(m[1]); e != "" {
				t.Errorf("%s: a diagram uses the HTML entity %s, which GitHub decodes before Mermaid parses it: write #%s", f, e, e[1:])
			}
		}
	}

	if diagrams < 10 {
		t.Fatalf("read %d diagrams, want at least 10: the docs moved and this test reads nothing", diagrams)
	}
}
