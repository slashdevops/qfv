package qfv

import (
	"slices"
	"strings"
	"testing"
)

// What PostgreSQL has and the SQL standard does not, each as an input the
// standard's grammar refuses, with what the refusal says.
var postgresOnly = []struct {
	input string
	says  string
}{
	// "<> is the standard SQL notation for not equal. != is an alias"
	{input: "id != 1", says: "!= is not part of the SQL:2023 dialect: write <>"},
	// "ILIKE ... is not in the SQL standard but is a PostgreSQL extension"
	{input: "name ILIKE 'x%'", says: `operator "ILIKE" is not part of the SQL:2023 dialect`},
	{input: "name NOT ILIKE 'x%'", says: `operator "ILIKE" is not part of the SQL:2023 dialect`},
	// "All of these operators are PostgreSQL-specific": ~~ ~~* !~~ !~~*
	{input: "name ~~ 'x%'", says: "~~ is not part of the SQL:2023 dialect: write LIKE"},
	{input: "name !~~ 'x%'", says: "!~~ is not part of the SQL:2023 dialect: write NOT LIKE"},
	{input: "name ~~* 'x%'", says: `operator "ILIKE" is not part of the SQL:2023 dialect`},
	{input: "name !~~* 'x%'", says: `operator "ILIKE" is not part of the SQL:2023 dialect`},
	// The standard's regular expressions are LIKE_REGEX, not ~.
	{input: "name ~ '^x'", says: `operator "~" is not part of the SQL:2023 dialect`},
	{input: "name !~* '^x'", says: `operator "~" is not part of the SQL:2023 dialect`},
	// "ISNULL ... NOTNULL ... (nonstandard syntax)"
	{input: "name ISNULL", says: "ISNULL is not part of the SQL:2023 dialect: write IS NULL"},
	{input: "name NOTNULL", says: "NOTNULL is not part of the SQL:2023 dialect: write IS NOT NULL"},
	// "a zero-length ESCAPE value is disallowed" by the standard
	{input: "name LIKE 'x%' ESCAPE ''", says: "an empty ESCAPE is not part of the SQL:2023 dialect"},
	{input: "name SIMILAR TO 'x%' ESCAPE ''", says: "an empty ESCAPE is not part of the SQL:2023 dialect"},
	// Inside an expression, the refusal is the same.
	{input: "id = 1 AND (name ILIKE 'x%' OR id = 2)", says: `operator "ILIKE" is not part of the SQL:2023 dialect`},
}

// What both have: every predicate the parser knows, in its standard spelling.
var inTheStandard = []string{
	"id = 1", "id <> 1", "id < 1", "id <= 1", "id > 1", "id >= 1", // E061-01
	"id BETWEEN 1 AND 2", "id NOT BETWEEN 1 AND 2", // E061-02
	"id BETWEEN SYMMETRIC 2 AND 1", "id BETWEEN ASYMMETRIC 1 AND 2", // T461
	"id IN (1, 2)", "id NOT IN (1, 2)", // E061-03
	"name LIKE 'x%'", "name NOT LIKE 'x%'", // E061-04
	"name LIKE 'x!%' ESCAPE '!'", "name NOT LIKE 'x!%' ESCAPE '!'", // E061-05
	"name IS NULL", "name IS NOT NULL", // E061-06
	"name SIMILAR TO '(x|y)%'", "name NOT SIMILAR TO 'x!%' ESCAPE '!'", // T141
	"id IS DISTINCT FROM 1", "id IS NOT DISTINCT FROM 1", // T151, T152
	"active IS TRUE", "active IS NOT FALSE", "active IS UNKNOWN", // F571
	"active = TRUE", "price = 1.5", "price = -2", "name = 'it''s'", // literals
	"id = 1 AND (name LIKE 'x%' OR NOT active IS TRUE)", // AND, OR, NOT, a group
}

func TestTheDefaultDialectIsPostgreSQL(t *testing.T) {
	t.Parallel()

	var zero Dialect
	if zero != DialectPostgreSQL || zero.String() != "PostgreSQL" {
		t.Errorf("the zero Dialect is %q, want PostgreSQL", zero)
	}

	if got := DialectPostgreSQL.Operators(); !slices.Equal(got, AllOperators()) {
		t.Errorf("PostgreSQL's operators are %v, want every operator", got)
	}

	allowed := []string{"id", "name", "price", "active"}

	// Without the option and with it, what PostgreSQL has is accepted, as it
	// was before there were dialects.
	for _, parser := range map[string]*FilterParser{
		"no option":             NewFilterParser(allowed),
		"WithDialect(Postgres)": NewFilterParser(allowed, WithDialect(DialectPostgreSQL)),
	} {
		for _, tt := range postgresOnly {
			if _, err := parser.Parse(tt.input); err != nil {
				t.Errorf("Parse(%q): %v, want it accepted in PostgreSQL", tt.input, err)
			}
		}
	}
}

func TestTheSQLStandardDialect(t *testing.T) {
	t.Parallel()

	allowed := []string{"id", "name", "price", "active"}
	parser := NewFilterParser(allowed, WithDialect(DialectSQLStandard))

	if DialectSQLStandard.String() != "SQL:2023" {
		t.Errorf("String() = %q, want SQL:2023", DialectSQLStandard)
	}

	if got := DialectSQLStandard.Operators(); slices.Contains(got, OpILike) || slices.Contains(got, OpRegexMatch) || len(got) != len(AllOperators())-2 {
		t.Errorf("the standard's operators are %v, want every operator but ILIKE and ~", got)
	}

	for _, input := range inTheStandard {
		t.Run("accepted: "+input, func(t *testing.T) {
			t.Parallel()

			if _, err := parser.Parse(input); err != nil {
				t.Errorf("Parse(%q): %v, want it accepted: the standard has it", input, err)
			}
		})
	}

	for _, tt := range postgresOnly {
		t.Run("refused: "+tt.input, func(t *testing.T) {
			t.Parallel()

			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) accepted, want it refused: it is PostgreSQL's, not the standard's", tt.input)
			}

			if !strings.Contains(err.Error(), tt.says) {
				t.Errorf("Parse(%q) = %q, want it to say %q", tt.input, err.Error(), tt.says)
			}
		})
	}
}

// The dialect narrows the allowed operators: allowing ILIKE does not put it
// in the standard.
func TestTheDialectNarrowsTheAllowedOperators(t *testing.T) {
	t.Parallel()

	parser := NewFilterParser([]string{"name"}, WithAllowedOperators(OpILike, OpLike), WithDialect(DialectSQLStandard))

	if _, err := parser.Parse("name ILIKE 'x%'"); err == nil || !strings.Contains(err.Error(), "not part of the SQL:2023 dialect") {
		t.Errorf("ILIKE allowed in the standard dialect: %v", err)
	}

	if _, err := parser.Parse("name LIKE 'x%'"); err != nil {
		t.Errorf("LIKE refused: %v", err)
	}
}
