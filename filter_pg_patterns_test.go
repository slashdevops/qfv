package qfv

import (
	"reflect"
	"strings"
	"testing"
)

// What PostgreSQL's pattern operators take is in its manual, "9.7. Pattern
// Matching" (https://www.postgresql.org/docs/current/functions-matching.html):
//
//	string LIKE pattern [ESCAPE escape-character]
//	string SIMILAR TO pattern [ESCAPE escape-character]
//	text ~ text → boolean
//
// and "the parser actually translates LIKE et al." to the operators ~~, ~~*,
// !~~ and !~~*. A pattern is a string. Each row of this file was run against
// PostgreSQL 18.6 as `SELECT … WHERE (<input>)`, on a table with the columns
// id integer, name text, price numeric and active boolean; what it answered
// is in the comment.

// TestFilterParser_APatternIsAString: a number or a boolean after a pattern
// operator is "operator does not exist" (42883) for every column type, since
// no ~~ takes one, so the parser refuses it. The POSIX operators already
// did; LIKE, ILIKE, SIMILAR TO and the ~~ family took any literal.
func TestFilterParser_APatternIsAString(t *testing.T) {
	allowed := []string{"id", "name", "price", "active"}

	refused := []struct {
		name  string
		input string
	}{
		// operator does not exist: text ~~ integer
		{name: "LIKE an integer", input: "name LIKE 1"},
		{name: "LIKE a negative integer", input: "name LIKE -1"},
		// operator does not exist: text !~~ numeric
		{name: "NOT LIKE a float", input: "name NOT LIKE 1.5"},
		// operator does not exist: text ~~* integer
		{name: "ILIKE an integer", input: "name ILIKE 1"},
		// operator does not exist: text !~~* integer
		{name: "NOT ILIKE an integer", input: "name NOT ILIKE 1"},
		// operator does not exist: text ~~ integer, and so on
		{name: "~~ an integer", input: "name ~~ 1"},
		{name: "~~* an integer", input: "name ~~* 1"},
		{name: "!~~ an integer", input: "name !~~ 1"},
		{name: "!~~* an integer", input: "name !~~* 1"},
		// function pg_catalog.similar_to_escape(integer) does not exist
		{name: "SIMILAR TO an integer", input: "name SIMILAR TO 1"},
		{name: "NOT SIMILAR TO an integer", input: "name NOT SIMILAR TO 1"},
		// operator does not exist: text ~~ boolean
		{name: "LIKE a boolean", input: "name LIKE TRUE"},
		// function pg_catalog.similar_to_escape(boolean) does not exist
		{name: "SIMILAR TO a boolean", input: "name SIMILAR TO false"},
		// Not for the column's type: integer ~~ integer, numeric ~~ integer
		// and boolean ~~ boolean do not exist either.
		{name: "an integer column LIKE an integer", input: "id LIKE 1"},
		{name: "a numeric column LIKE an integer", input: "price LIKE 1"},
		{name: "a boolean column LIKE a boolean", input: "active LIKE true"},
		// In the middle of an expression, and negated.
		{name: "inside a group", input: "id = 1 AND (name LIKE 2 OR id = 3)"},
		{name: "under NOT", input: "NOT name ILIKE 2"},
		// The ESCAPE clause is not part of this grammar.
		{name: "ESCAPE", input: "name LIKE 'x!%' ESCAPE '!'"},
		{name: "SIMILAR TO with ESCAPE", input: "name SIMILAR TO 'x!%' ESCAPE '!'"},
	}

	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) = %v, want an error: PostgreSQL refuses it", tt.input, node)
			}
		})
	}

	// The refusal says what was expected, and names the operator as it is
	// written in SQL.
	for input, operator := range map[string]string{
		"name LIKE 1":           "LIKE",
		"name NOT LIKE 1":       "NOT LIKE",
		"name ILIKE 1":          "ILIKE",
		"name NOT ILIKE 1":      "NOT ILIKE",
		"name SIMILAR TO 1":     "SIMILAR TO",
		"name NOT SIMILAR TO 1": "NOT SIMILAR TO",
	} {
		t.Run("says so: "+input, func(t *testing.T) {
			_, err := NewFilterParser(allowed).Parse(input)
			if err == nil {
				t.Fatalf("Parse(%q): no error", input)
			}

			if want := "expected string pattern for operator " + operator + ","; !strings.Contains(err.Error(), want) {
				t.Errorf("Parse(%q) = %q, want it to say %q", input, err.Error(), want)
			}
		})
	}

	// A string is a pattern, whatever is in it: PostgreSQL runs each of these.
	for _, input := range []string{
		"name LIKE 'x%'",
		"name NOT LIKE '_x'",
		"name ILIKE 'X%'",
		"name NOT ILIKE 'X%'",
		"name ~~ 'x%'",
		"name ~~* 'X%'",
		"name !~~ 'x%'",
		"name !~~* 'X%'",
		"name SIMILAR TO '(x|y)%'",
		"name NOT SIMILAR TO '(x|y)%'",
		// A number written as a string is a pattern.
		"name LIKE '1'",
		"name LIKE '-1'",
		// The default escape character, the backslash.
		`name LIKE '100\%'`,
		"name LIKE ''",
	} {
		t.Run("accepted: "+input, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}

			var pattern Node

			switch n := node.(type) {
			case *BinaryOperatorNode:
				pattern = n.Right
			case *SimilarToNode:
				pattern = n.Pattern
			default:
				t.Fatalf("Parse(%q) = %T, want a pattern node", input, node)
			}

			if literal, ok := pattern.(*LiteralNode); !ok || literal.Kind != reflect.String {
				t.Errorf("Parse(%q): the pattern is %v, want a string literal", input, pattern)
			}
		})
	}
}

// TestFilterParser_IsIsAKeywordInAnyCase: PostgreSQL folds an unquoted word,
// so `id is null` is `id IS NULL`. The lexer kept the lower-case word back as
// an identifier, so every other spelling of IS was the keyword and that one
// was refused. PostgreSQL answers each of these with rows.
func TestFilterParser_IsIsAKeywordInAnyCase(t *testing.T) {
	allowed := []string{"id", "active"}

	for input, want := range map[string]string{
		"id is null":                "id IS NULL",
		"id IS NULL":                "id IS NULL",
		"id Is Not Null":            "id IS NOT NULL",
		"id is not null":            "id IS NOT NULL",
		"id is distinct from 1":     "id IS DISTINCT FROM 1",
		"id is not distinct from 1": "id IS NOT DISTINCT FROM 1",
		"active is true":            "active IS TRUE",
		"active is not unknown":     "active IS NOT UNKNOWN",
	} {
		t.Run(input, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}

			if got := node.String(); got != want {
				t.Errorf("Parse(%q) = %s, want %s", input, got, want)
			}
		})
	}

	// And so `is` is no name, as it is none in PostgreSQL: with a column of
	// that name allowed, `is = 1` is a syntax error at or near "=".
	if node, err := NewFilterParser([]string{"is", "id"}).Parse("is = 1"); err == nil {
		t.Errorf(`Parse("is = 1") = %v, want an error: IS is reserved`, node)
	}
}

// TestFilterParser_YesAndNoAreNames: SQL has two boolean literals, TRUE and
// FALSE. YES and NO were read as booleans here, and PostgreSQL reads them as
// names: `active = YES` compares with a column called yes (and where there is
// none, "column yes does not exist"). The grammar compares a field with a
// literal, so it is refused; and a field called yes or no is one.
func TestFilterParser_YesAndNoAreNames(t *testing.T) {
	allowed := []string{"id", "active"}

	for _, input := range []string{
		"active = YES",
		"active = no",
		"active <> NO",
		"active IN (yes, no)",
		"active IS DISTINCT FROM yes",
		"id BETWEEN yes AND no",
		// syntax error at or near "yes": IS takes TRUE, FALSE, UNKNOWN, NULL.
		"active IS yes",
		"active IS NOT NO",
	} {
		t.Run(input, func(t *testing.T) {
			if node, err := NewFilterParser(allowed).Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %v, want an error: YES and NO are no literals", input, node)
			}
		})
	}

	// TRUE and FALSE, in any case, are the literals.
	for input, want := range map[string]bool{"active = TRUE": true, "active = true": true, "active = False": false, "active <> FALSE": false} {
		t.Run(input, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}

			literal, ok := node.(*BinaryOperatorNode).Right.(*LiteralNode)
			if !ok || literal.Kind != reflect.Bool || literal.Value != want {
				t.Errorf("Parse(%q): the value is %v, want the boolean %v", input, node.(*BinaryOperatorNode).Right, want)
			}
		})
	}

	// A column called yes or no is a field like any other. PostgreSQL runs
	// `yes = 1 AND no = 2` on a table that has them.
	node, err := NewFilterParser([]string{"yes", "no"}).Parse("yes = 1 AND no = 2")
	if err != nil {
		t.Fatalf(`a field called yes: %v`, err)
	}

	if got, want := node.String(), "((yes = 1) AND (no = 2))"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	// And a string that says yes is a string: PostgreSQL reads 'yes' as a
	// boolean's input, which is the caller's column to know.
	if _, err := NewFilterParser(allowed).Parse("active = 'yes'"); err != nil {
		t.Errorf(`active = 'yes': %v`, err)
	}
}
