package qfv

import (
	"strings"
	"testing"
)

func TestFilterParser_OperatorGating(t *testing.T) {
	fields := []string{"a", "b", "name", "age"}

	t.Run("default allows everything", func(t *testing.T) {
		p := NewFilterParser(fields)
		for _, in := range []string{
			"a = 1", "a LIKE 'x'", "a ILIKE 'x'", "a IN (1, 2)", "a BETWEEN 1 AND 2",
			"a IS NULL", "a IS TRUE", "a IS DISTINCT FROM 1", "a ~ 'x'", "a SIMILAR TO 'x'",
			"NOT (a = 1)", "a = 1 AND b = 2", "a = 1 OR b = 2",
		} {
			if _, err := p.Parse(in); err != nil {
				t.Errorf("default parser rejected %q: %v", in, err)
			}
		}
	})

	t.Run("restrict to comparison + logical", func(t *testing.T) {
		p := NewFilterParser(fields, WithAllowedOperatorGroups(GroupComparison, GroupLogical))
		allowed := []string{"a = 1", "a <> 1", "a > 1 AND b < 2", "NOT (a = 1)", "a = 1 OR b = 2"}
		for _, in := range allowed {
			if _, err := p.Parse(in); err != nil {
				t.Errorf("expected %q allowed: %v", in, err)
			}
		}
		blocked := map[string]string{
			"a LIKE 'x'":           `"LIKE"`,
			"a IN (1, 2)":          `"IN"`,
			"a IS NULL":            `"IS NULL"`,
			"a IS TRUE":            `"IS BOOLEAN"`,
			"a IS DISTINCT FROM 1": `"IS DISTINCT FROM"`,
			"a ~ 'x'":              `"~"`,
			"a BETWEEN 1 AND 2":    `"BETWEEN"`,
		}
		for in, want := range blocked {
			_, err := p.Parse(in)
			if err == nil {
				t.Errorf("expected %q blocked", in)
				continue
			}
			if !strings.Contains(err.Error(), "is not allowed") || !strings.Contains(err.Error(), want) {
				t.Errorf("Parse(%q) error = %v, want mention of %s not allowed", in, err, want)
			}
		}
	})

	t.Run("individual operators are additive with groups", func(t *testing.T) {
		p := NewFilterParser(fields,
			WithAllowedOperatorGroups(GroupComparison),
			WithAllowedOperators(OpIn, OpAnd))
		if _, err := p.Parse("a = 1 AND a IN (1, 2)"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if _, err := p.Parse("a LIKE 'x'"); err == nil {
			t.Error("LIKE should still be blocked")
		}
	})

	t.Run("negated forms follow the base operator", func(t *testing.T) {
		p := NewFilterParser(fields, WithAllowedOperators(OpIn, OpLike, OpBetween))
		for _, in := range []string{"a NOT IN (1, 2)", "a NOT LIKE 'x'", "a NOT BETWEEN 1 AND 2"} {
			if _, err := p.Parse(in); err != nil {
				t.Errorf("expected %q allowed via base operator: %v", in, err)
			}
		}
	})

	t.Run("standalone NOT gated by OpNot", func(t *testing.T) {
		// OpEqual allowed but OpNot not: the leading NOT should be rejected.
		p := NewFilterParser(fields, WithAllowedOperators(OpEqual))
		if _, err := p.Parse("NOT (a = 1)"); err == nil {
			t.Error("expected leading NOT to be blocked without OpNot")
		}
	})
}

func TestOperatorGroups_Coverage(t *testing.T) {
	// Every operator returned by AllOperators must belong to exactly one group,
	// guarding against a new operator being added without a group.
	inGroup := map[Operator]int{}
	for _, g := range []OperatorGroup{
		GroupLogical, GroupComparison, GroupPattern, GroupMembership,
		GroupRange, GroupNull, GroupBoolean, GroupDistinct,
	} {
		for _, op := range g.Operators() {
			inGroup[op]++
		}
	}
	for _, op := range AllOperators() {
		if inGroup[op] != 1 {
			t.Errorf("operator %q appears in %d groups, want exactly 1", op, inGroup[op])
		}
	}
}

func TestSortParser_DirectionGating(t *testing.T) {
	fields := []string{"name", "created_at"}

	t.Run("default allows ASC and DESC", func(t *testing.T) {
		p := NewSortParser(fields)
		if _, err := p.Parse("name ASC, created_at DESC"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("restrict to ASC only", func(t *testing.T) {
		p := NewSortParser(fields, WithAllowedDirections(SortAsc))
		if _, err := p.Parse("name ASC"); err != nil {
			t.Errorf("ASC should be allowed: %v", err)
		}
		_, err := p.Parse("name DESC")
		if err == nil {
			t.Fatal("DESC should be blocked")
		}
		if !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("error = %v, want 'not allowed'", err)
		}
	})
}

func TestFilterParser_DottedIdentifiers(t *testing.T) {
	p := NewFilterParser([]string{"q.v.last_name", "user.profile.age", "name"})

	tests := []struct {
		in      string
		wantStr string
	}{
		{"q.v.last_name = 'Doe'", "(q.v.last_name = 'Doe')"},
		{"user.profile.age >= 18 AND name = 'x'", "((user.profile.age >= 18) AND (name = 'x'))"},
		{"q.v.last_name IN ('a', 'b')", "q.v.last_name IN ('a', 'b')"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			node, err := p.Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.in, err)
			}
			if got := node.String(); got != tt.wantStr {
				t.Errorf("String() = %q, want %q", got, tt.wantStr)
			}
		})
	}

	t.Run("unknown dotted field rejected", func(t *testing.T) {
		if _, err := p.Parse("q.v.secret = 'x'"); err == nil {
			t.Error("expected unknown dotted field to be rejected")
		}
	})

	t.Run("numeric literals still parse", func(t *testing.T) {
		fp := NewFilterParser([]string{"score", "age"})
		if _, err := fp.Parse("score = 3.14 AND age = 42"); err != nil {
			t.Errorf("numeric literals broke: %v", err)
		}
	})
}
