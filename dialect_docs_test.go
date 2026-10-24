package qfv

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// docs/dialects.md quotes, word for word, what Parse answers in each dialect.
// A message that changes here and not there would leave the page telling
// clients to look for an error they never get, so each row of its tables is
// parsed and held to the parser: the "### SQL:2023" table in the SQL standard
// dialect, the "### MySQL 8.4" one in MySQL's. A row that names an allow-list
// ("with the allow-list `["name"]`") is parsed with it.
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

	row := regexp.MustCompile("(?m)^\\| (`[^|]+`(?:, `[^|]+`)*) \\| `([^`]+)`(?: \\(with the allow-list `\\[([^]]*)\\]`\\))? \\|$")
	code := regexp.MustCompile("`([^`]+)`")
	quoted := regexp.MustCompile(`"([^"]+)"`)

	dialects := map[string]Dialect{"SQL:2023": DialectSQLStandard, "MySQL 8.4": DialectMySQL}
	checked := map[string]int{}

	for _, table := range strings.Split(section, "\n### ")[1:] {
		heading, body, _ := strings.Cut(table, "\n")

		dialect, ok := dialects[heading]
		if !ok {
			t.Errorf("docs/dialects.md has a refusals table for %q, a dialect this test does not know", heading)

			continue
		}

		for _, m := range row.FindAllStringSubmatch(body, -1) {
			allowed := []string{"id", "name"}
			if m[3] != "" {
				allowed = nil
				for _, f := range quoted.FindAllStringSubmatch(m[3], -1) {
					allowed = append(allowed, f[1])
				}
			}

			parser := NewFilterParser(allowed, WithDialect(dialect))

			for _, input := range code.FindAllStringSubmatch(m[1], -1) {
				checked[heading]++

				_, err := parser.Parse(input[1])
				if err == nil {
					t.Errorf("%s: the page says %q is refused, and Parse accepts it", heading, input[1])

					continue
				}

				if !strings.Contains(err.Error(), m[2]) {
					t.Errorf("%s: the page says %q answers %q; Parse answers %q", heading, input[1], m[2], err.Error())
				}
			}
		}
	}

	if checked["SQL:2023"] < 9 || checked["MySQL 8.4"] < 12 {
		t.Fatalf("read %v inputs from the page's tables, want at least 9 for SQL:2023 and 12 for MySQL 8.4: the tables' shape changed and this test reads nothing", checked)
	}
}
