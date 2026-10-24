package qfv

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// docs/dialects.md quotes, word for word, what Parse answers in the SQL:2023
// dialect. A message that changes here and not there would leave the page
// telling clients to look for an error they never get, so each row of its
// table is parsed and held to the parser.
func TestTheDialectsPageQuotesTheParser(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("docs/dialects.md")
	if err != nil {
		t.Fatal(err)
	}

	page := string(raw)

	start := strings.Index(page, "## What the refusals say")
	if start < 0 {
		t.Fatal(`docs/dialects.md has no "What the refusals say" section`)
	}

	section := page[start:]
	if end := strings.Index(section[1:], "\n## "); end >= 0 {
		section = section[:end+1]
	}

	row := regexp.MustCompile("(?m)^\\| (`[^|]+`(?:, `[^|]+`)*) \\| `([^`]+)` \\|$")
	code := regexp.MustCompile("`([^`]+)`")
	parser := NewFilterParser([]string{"id", "name"}, WithDialect(DialectSQLStandard))

	checked := 0

	for _, m := range row.FindAllStringSubmatch(section, -1) {
		for _, input := range code.FindAllStringSubmatch(m[1], -1) {
			checked++

			_, err := parser.Parse(input[1])
			if err == nil {
				t.Errorf("the page says %q is refused, and Parse accepts it", input[1])

				continue
			}

			if !strings.Contains(err.Error(), m[2]) {
				t.Errorf("the page says %q answers %q; Parse answers %q", input[1], m[2], err.Error())
			}
		}
	}

	if checked < 9 {
		t.Fatalf("read %d inputs from the page's table, want at least 9: the table's shape changed and this test reads nothing", checked)
	}
}
