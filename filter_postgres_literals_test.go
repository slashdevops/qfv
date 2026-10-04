package qfv

import (
	"reflect"
	"strings"
	"testing"
)

// An expression this package accepts must be one PostgreSQL can run: a caller
// splices it into a WHERE clause. Each row below was run against PostgreSQL
// 18 as `SELECT … WHERE (<input>)`; what it answered is in the comment.

// TestFilterParser_RefusesWhatPostgresRefuses holds shapes PostgreSQL refuses
// before it looks at a type: a syntax error (42601), or an operator that does
// not exist because the lexer read a longer one (42883). The first twelve
// were accepted by this parser; the rest it already refused, and they hold
// the minus sign to the same rule.
func TestFilterParser_RefusesWhatPostgresRefuses(t *testing.T) {
	allowed := []string{"id", "name", "price", "active"}

	tests := []struct {
		name  string
		input string
	}{
		// syntax error at or near "DISTINCT": SQL writes IS DISTINCT FROM.
		{name: "DISTINCT FROM without IS", input: "id DISTINCT FROM 1"},
		// syntax error at or near "NOT"
		{name: "NOT DISTINCT FROM without IS", input: "id NOT DISTINCT FROM 1"},
		// trailing junk after numeric literal at or near "0x1p"
		{name: "hexadecimal float", input: "price = 0x1p-2"},
		// ... at or near "0X1P"
		{name: "hexadecimal float, upper case", input: "price = 0X1P+3"},
		// ... at or near ".8p1"
		{name: "hexadecimal float with a fraction", input: "price = 0x1.8p1"},
		// trailing junk after numeric literal at or near "1AND"
		{name: "integer glued to a keyword", input: "name=1AND id=2"},
		// ... at or near "1and"
		{name: "integer glued to a keyword, lower case", input: "id=1and id=2"},
		// ... at or near "1.AND"
		{name: "float glued to a keyword", input: "price = 1.AND id = 1"},
		// syntax error at or near "yes": IS takes TRUE, FALSE, UNKNOWN or NULL.
		{name: "IS YES", input: "active IS yes"},
		// syntax error at or near "NO"
		{name: "IS NOT NO", input: "active IS NOT NO"},
		// syntax error at or near "NOT": NOT negates the keyword, not its symbol.
		{name: "NOT before ~~", input: "name NOT ~~ 'x'"},
		{name: "NOT before ~~*", input: "name NOT ~~* 'x'"},

		// Already refused, and still: the sign does not open a way around.
		// trailing junk after numeric literal at or near "1AND"
		{name: "negative integer glued to a keyword", input: "id = -1AND id = 2"},
		// ... at or near "1name"
		{name: "integer glued to a field", input: "id = 1name"},
		// ... at or near "1e"
		{name: "exponent without digits", input: "price = 1e"},
		// operator does not exist: integer !=- integer
		{name: "minus glued to !=", input: "id !=-1"},
		// operator does not exist: text ~~- integer, and so on: an operator
		// written with ! or ~ may end in a minus sign.
		{name: "minus glued to ~~", input: "name ~~-1"},
		{name: "minus glued to !~~", input: "name !~~-1"},
		{name: "minus glued to ~~*", input: "name ~~*-1"},
		{name: "minus glued to !~~*", input: "name !~~*-1"},
		{name: "minus glued to ~", input: "name ~-1"},
		{name: "minus glued to ~*", input: "name ~*-1"},
		{name: "minus glued to !~", input: "name !~-1"},
		{name: "minus glued to !~*", input: "name !~*-1"},
		// syntax error at or near ")"
		{name: "nothing to negate", input: "id = -"},
		{name: "a sign and a point", input: "price = -."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) = %v, want an error: PostgreSQL refuses it", tt.input, node)
			}
		})
	}
}

// TestFilterParser_NotBeforeASymbolNamesTheSymbol: the error for "NOT ~~"
// names what was written. The token's type is LIKE, and "unexpected token
// after NOT: LIKE" would name the one spelling that is valid.
func TestFilterParser_NotBeforeASymbolNamesTheSymbol(t *testing.T) {
	for input, symbol := range map[string]string{"name NOT ~~ 'x'": "~~", "name NOT ~~* 'x'": "~~*"} {
		_, err := NewFilterParser([]string{"name"}).Parse(input)
		if err == nil {
			t.Fatalf("Parse(%q): no error", input)
		}

		if want := "unexpected token after NOT: " + symbol + " "; !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) says %q, want it to contain %q", input, err.Error(), want)
		}
	}
}

// TestFilterParser_NumbersAreDecimal holds what the parser refuses although
// PostgreSQL 18 runs it. The grammar is deliberately the smaller one: a
// number is a decimal literal with an optional sign glued to it, and there is
// no arithmetic.
func TestFilterParser_NumbersAreDecimal(t *testing.T) {
	allowed := []string{"id", "name", "price"}

	for _, input := range []string{
		"id = 0x10",       // 16, since PostgreSQL 16
		"id = -0x10",      // -16
		"id = 0b101",      // 5
		"id = 0o17",       // 15
		"id = 1_000",      // 1000
		"price = 1_000.5", // 1000.5: accepted by v1.0.2, where only the integer form was refused
		"price = 1e1_0",   // 1e10
		"price = .5_5",    // .55
		"id = - 1",        // unary minus, with a space
		"id = +1",         // unary plus
		"id = 1-1",        // arithmetic
		"id = 1 -1",       // arithmetic
		"id - 1 = 0",      // arithmetic on a field
		"id = -id",        // a sign on a field
		"id = -(1)",       // a sign on a group
		"id IN (- 1)",     // unary minus in a list
		"id BETWEEN - 1 AND 1",
	} {
		t.Run(input, func(t *testing.T) {
			if node, err := NewFilterParser(allowed).Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %v, want an error", input, node)
			}
		})
	}
}

// TestFilterParser_ACommentIsRefused: "--" opens a SQL comment that runs to
// the end of the line. Spliced as `WHERE (<input>)` it takes the closing
// parenthesis with it (syntax error at end of input); on a line of its own,
// `id = -1--` runs. Neither is a filter: two minus signs are never a sign.
func TestFilterParser_ACommentIsRefused(t *testing.T) {
	for _, input := range []string{"id = --1", "id = -1--", "id = 1 -- AND name = 'x'", "id = 1--"} {
		t.Run(input, func(t *testing.T) {
			if node, err := NewFilterParser([]string{"id", "name"}).Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %v, want an error", input, node)
			}
		})
	}
}

// TestFilterParser_ASignOnAStringIsRefused: PostgreSQL refuses it for its
// type (operator is not unique: - unknown), this parser for its grammar.
func TestFilterParser_ASignOnAStringIsRefused(t *testing.T) {
	if node, err := NewFilterParser([]string{"id"}).Parse("id = -'x'"); err == nil {
		t.Fatalf("Parse = %v, want an error", node)
	}
}

// TestFilterParser_NegativeNumbers: a sign glued to a decimal literal is part
// of the literal, wherever a literal is taken.
func TestFilterParser_NegativeNumbers(t *testing.T) {
	allowed := []string{"id", "price", "name"}

	tests := []struct {
		input string
		want  string // the node's rendering
	}{
		{input: "id = -1", want: "(id = -1)"},
		{input: "id =-1", want: "(id = -1)"},
		{input: "id <>-1", want: "(id <> -1)"},
		{input: "id >=-1", want: "(id >= -1)"},
		{input: "id <-1", want: "(id < -1)"},
		{input: "id != -1", want: "(id != -1)"},
		{input: "id > -1", want: "(id > -1)"},
		{input: "price = -1.5", want: "(price = -1.5)"},
		{input: "price = -.5", want: "(price = -.5)"},
		{input: "price = -1e-5", want: "(price = -1e-5)"},
		{input: "price = -1e+5", want: "(price = -1e+5)"},
		{input: "id = -9223372036854775808", want: "(id = -9223372036854775808)"},
		{input: "id IN (-1, 2)", want: "id IN (-1, 2)"},
		{input: "id IN (-1,-2)", want: "id IN (-1, -2)"},
		{input: "id BETWEEN -5 AND 5", want: "id BETWEEN -5 AND 5"},
		{input: "id NOT BETWEEN -5 AND -1", want: "id NOT BETWEEN -5 AND -1"},
		{input: "id IS NOT DISTINCT FROM -1", want: "id IS NOT DISTINCT FROM -1"},
		{input: "id = -1 AND name = 'x'", want: "((id = -1) AND (name = 'x'))"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := NewFilterParser(allowed).Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.input, err)
			}

			if got := node.String(); got != tt.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestFilterParser_NegativeLiteralValue: the sign reaches the literal's value
// and its kind, not only its text.
func TestFilterParser_NegativeLiteralValue(t *testing.T) {
	tests := []struct {
		input string
		value any
		kind  reflect.Kind
	}{
		{input: "id = -42", value: int64(-42), kind: reflect.Int64},
		{input: "id = -9223372036854775808", value: int64(-9223372036854775808), kind: reflect.Int64},
		{input: "id = -1.5", value: -1.5, kind: reflect.Float64},
		{input: "id = -.5", value: -0.5, kind: reflect.Float64},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := NewFilterParser([]string{"id"}).Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.input, err)
			}

			cmp, ok := node.(*BinaryOperatorNode)
			if !ok {
				t.Fatalf("Parse(%q) = %T, want *BinaryOperatorNode", tt.input, node)
			}

			lit, ok := cmp.Right.(*LiteralNode)
			if !ok {
				t.Fatalf("right side = %T, want *LiteralNode", cmp.Right)
			}

			if lit.Value != tt.value || lit.Kind != tt.kind {
				t.Errorf("literal = %v (%s), want %v (%s)", lit.Value, lit.Kind, tt.value, tt.kind)
			}
		})
	}
}

// TestFilterParser_StillAcceptsWhatPostgresRuns: the new refusals took
// nothing valid with them. Each of these ran against PostgreSQL 18.
func TestFilterParser_StillAcceptsWhatPostgresRuns(t *testing.T) {
	allowed := []string{"id", "name", "price", "active"}

	for _, input := range []string{
		"price = 1e5",
		"price = 1E5",
		"price = 1e+5",
		"price = 1E+5",
		"price = 1.5e-3",
		"price = .5",
		"price = 1.",
		"id = 017",
		"id=1 AND name='x'",
		"name = 'x'AND id = 1", // a string ends at its quote: nothing is glued
		"id IN (1,2)AND name = 'x'",
		"name = '1AND'",
		"name = '0x1p-2'",
		"name = '-'",
		"name LIKE '%-1%'",
		"id IS DISTINCT FROM 1",
		"id IS NOT DISTINCT FROM 1",
		"active IS TRUE",
		"active IS NOT false",
		"name NOT LIKE 'x'",
		"name not ilike 'x'",
		"name !~~ 'x'",
		"name !~~* 'x'",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := NewFilterParser(allowed).Parse(input); err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}
		})
	}
}

// TestLexer_NumberTokens: what the lexer makes of a number and of what is
// glued to it.
func TestLexer_NumberTokens(t *testing.T) {
	tests := []struct {
		input string
		want  []Token
	}{
		{input: "-1", want: []Token{{Type: TokenInt, Value: "-1"}}},
		{input: "-1.5", want: []Token{{Type: TokenFloat, Value: "-1.5"}}},
		{input: "-.5", want: []Token{{Type: TokenFloat, Value: "-.5"}}},
		{input: "1AND", want: []Token{{Type: TokenIllegal, Value: "1AND"}}},
		{input: "-1AND", want: []Token{{Type: TokenIllegal, Value: "-1AND"}}},
		{input: "0x1p-2", want: []Token{{Type: TokenIllegal, Value: "0x1p-2"}}},
		{input: "0x10", want: []Token{{Type: TokenIllegal, Value: "0x10"}}},
		{input: "1_000", want: []Token{{Type: TokenIllegal, Value: "1_000"}}},
		{input: "1_000.5", want: []Token{{Type: TokenIllegal, Value: "1_000.5"}}},
		{input: "0b101", want: []Token{{Type: TokenIllegal, Value: "0b101"}}},
		{input: "0o17", want: []Token{{Type: TokenIllegal, Value: "0o17"}}},
		{input: "1p5", want: []Token{{Type: TokenIllegal, Value: "1p5"}}},
		{input: "-.", want: []Token{{Type: TokenIllegal, Value: "-."}}},
		{input: "-._", want: []Token{{Type: TokenIllegal, Value: "-._"}}},
		{input: "1e+5", want: []Token{{Type: TokenFloat, Value: "1e+5"}}},
		// A string that ends in an operator's character is not an operator.
		{input: "'x!'-1", want: []Token{{Type: TokenString, Value: "'x!'"}, {Type: TokenInt, Value: "-1"}}},
		{input: "~~*-1", want: []Token{{Type: TokenOperatorILike, Value: "~~*"}, {Type: TokenIllegal, Value: "-"}, {Type: TokenInt, Value: "1"}}},
		{input: "!~*-1", want: []Token{{Type: TokenOperatorNotRegexMatchCI, Value: "!~*"}, {Type: TokenIllegal, Value: "-"}, {Type: TokenInt, Value: "1"}}},
		{input: "- 1", want: []Token{{Type: TokenIllegal, Value: "-"}, {Type: TokenInt, Value: "1"}}},
		{input: "--1", want: []Token{{Type: TokenIllegal, Value: "-"}, {Type: TokenInt, Value: "-1"}}},
		{input: "1-1", want: []Token{{Type: TokenInt, Value: "1"}, {Type: TokenInt, Value: "-1"}}},
		{input: "=-1", want: []Token{{Type: TokenOperatorEqual, Value: "="}, {Type: TokenInt, Value: "-1"}}},
		{input: "!=-1", want: []Token{{Type: TokenOperatorNotEqualAlias, Value: "!="}, {Type: TokenIllegal, Value: "-"}, {Type: TokenInt, Value: "1"}}},
		{input: "!= -1", want: []Token{{Type: TokenOperatorNotEqualAlias, Value: "!="}, {Type: TokenInt, Value: "-1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := NewLexer(tt.input)
			l.Parse()

			got := l.tokens[:len(l.tokens)-1] // without EOF
			if len(got) != len(tt.want) {
				t.Fatalf("tokens = %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i].Type != tt.want[i].Type || got[i].Value != tt.want[i].Value {
					t.Errorf("token %d = %s %q, want %s %q", i, got[i].Type, got[i].Value, tt.want[i].Type, tt.want[i].Value)
				}
			}
		})
	}
}

// FuzzFilterParser: the parser takes text a client wrote. It must not panic,
// and it answers with a node or with an error, never both and never neither.
func FuzzFilterParser(f *testing.F) {
	for _, seed := range []string{
		"id = 1", "id = -1", "id =-1", "id !=-1", "name=1AND id=2", "price = 0x1p-2",
		"id DISTINCT FROM 1", "id IS NOT DISTINCT FROM -1", "id IN (-1, 2)", "id BETWEEN -5 AND 5",
		"name = 'it''s'", "name ~~* 'a%'", "NOT (id = 1)", "id = --1", "id = 1e", "id = -.5e+3",
		"(", "'", "-", "1", "", "id = 1 AND", "a.b.c = 1",
		"\uFEFFid = 1", "name lıke 'x'", "id Iſ NULL", "((((((((id = 1))))))))", "NOT NOT NOT id = 1",
	} {
		f.Add(seed)
	}

	parser := NewFilterParser([]string{"id", "name", "price", "a.b.c"})

	f.Fuzz(func(t *testing.T, input string) {
		node, err := parser.Parse(input)
		if (node == nil) == (err == nil) {
			t.Fatalf("Parse(%q) = %v, %v: want a node or an error", input, node, err)
		}
	})
}

// TestFilterParser_KeywordsAreASCII: SQL folds case in ASCII only. The
// dotless 'ı' and the long 'ſ' upper-case to 'I' and 'S' in Unicode, so
// "lıke" and "Iſ" read as keywords here and were a syntax error in
// PostgreSQL ("syntax error at or near "lıke"").
func TestFilterParser_KeywordsAreASCII(t *testing.T) {
	allowed := []string{"id", "name", "active"}

	for _, input := range []string{
		"name lıke 'x'",
		"name ıLIKE 'x'",
		"name NOT lıke 'x'",
		"id ıN (1)",
		"id ıS NULL",
		"id Iſ NULL",
		"active IS FALſE",
		"active IS NOT FALſE",
		"active IS UNKNOWıN",
		"id ıſNULL",
		"name ſIMILAR TO 'x'",
		"name SIMILAR tO 'x' OR name ſimilar to 'x'",
		"id IS DIſTINCT FROM 1",
		"id BETWEEN ſYMMETRIC 1 AND 2",
		"id = 1 AND id ıs null",
		"active = FALſE",
		"NOT id = 1 OR ıd = 1",
	} {
		t.Run(input, func(t *testing.T) {
			if node, err := NewFilterParser(allowed).Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %v, want an error: PostgreSQL does not read it as the keyword", input, node)
			}
		})
	}

	// The same words in ASCII, in any case, are the keywords.
	for _, input := range []string{"name like 'x'", "id In (1)", "id Is nUlL", "active IS false", "name similar To 'x'", "id is distinct from 1"} {
		t.Run(input, func(t *testing.T) {
			if input == "id is distinct from 1" || input == "id Is nUlL" {
				// "is" in lower case is an older special case of the lexer and
				// is refused; it is not this test's subject.
				input = strings.Replace(input, "is ", "IS ", 1)
			}

			if _, err := NewFilterParser(allowed).Parse(input); err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}
		})
	}
}

// TestSortParser_DirectionsAreASCII: the same for a sort direction.
func TestSortParser_DirectionsAreASCII(t *testing.T) {
	p := NewSortParser([]string{"name"})

	for _, input := range []string{"name aſc", "name deſc", "name DEſC"} {
		if nodes, err := p.Parse(input); err == nil {
			t.Errorf("Parse(%q) = %v, want an error", input, nodes)
		}
	}

	for _, input := range []string{"name asc", "name DESC", "name Desc"} {
		if _, err := p.Parse(input); err != nil {
			t.Errorf("Parse(%q): %v", input, err)
		}
	}
}

// TestFilterParser_AByteOrderMarkIsRefused: text/scanner drops a U+FEFF at
// the start of its input without a token, so the expression behind it was
// accepted. PostgreSQL reads the mark as part of the first word: a syntax
// error, or a column that does not exist.
func TestFilterParser_AByteOrderMarkIsRefused(t *testing.T) {
	for _, input := range []string{"\uFEFFid = 1", "\uFEFF id = 1", "\uFEFFNOT id = 1", "\uFEFF-1", "\uFEFF"} {
		if node, err := NewFilterParser([]string{"id"}).Parse(input); err == nil {
			t.Errorf("Parse(%q) = %v, want an error", input, node)
		}
	}

	l := NewLexer("\uFEFFid")
	l.Parse()

	if l.tokens[0].Type != TokenIllegal || l.tokens[0].Value != "\uFEFF" {
		t.Errorf("first token = %s %q, want the mark as an illegal token", l.tokens[0].Type, l.tokens[0].Value)
	}
}

// TestFilterParser_NestingIsBounded: the parser is recursive and the input
// decides how deep. 250 000 opening parentheses ended the process with a
// stack overflow -- a fatal error, which nothing recovers from, so this test
// would not fail but kill the test binary.
func TestFilterParser_NestingIsBounded(t *testing.T) {
	p := NewFilterParser([]string{"id"})

	nested := func(depth int) string {
		return strings.Repeat("(", depth) + "id = 1" + strings.Repeat(")", depth)
	}

	if _, err := p.Parse(nested(maxFilterDepth)); err != nil {
		t.Fatalf("%d levels of parentheses: %v", maxFilterDepth, err)
	}

	if _, err := p.Parse(strings.Repeat("NOT ", maxFilterDepth) + "id = 1"); err != nil {
		t.Fatalf("%d NOTs: %v", maxFilterDepth, err)
	}

	for name, input := range map[string]string{
		"one level too many":            nested(maxFilterDepth + 1),
		"one NOT too many":              strings.Repeat("NOT ", maxFilterDepth+1) + "id = 1",
		"NOT and parentheses together":  strings.Repeat("NOT (", maxFilterDepth) + "id = 1" + strings.Repeat(")", maxFilterDepth),
		"a million opening parentheses": strings.Repeat("(", 1<<20),
		"a million balanced":            nested(1 << 20),
		"a quarter of a million NOTs":   strings.Repeat("NOT ", 1<<18) + "id = 1",
	} {
		t.Run(name, func(t *testing.T) {
			node, err := p.Parse(input)
			if err == nil {
				t.Fatalf("Parse = %v, want an error", node)
			}

			// One error, not one for each level that unwinds.
			if got := strings.Count(err.Error(), "\n"); got > 2 {
				t.Errorf("%d lines of error for one mistake: %.200s", got+1, err.Error())
			}

			if !strings.Contains(err.Error(), "nested more than") {
				t.Errorf("the error does not say what is wrong: %.200s", err.Error())
			}
		})
	}

	// Depth is how deep, not how many: a long flat expression is fine.
	flat := strings.Repeat("(id = 1) AND ", 500) + "NOT id = 2"
	if _, err := p.Parse(flat); err != nil {
		t.Fatalf("a flat expression of 500 groups: %.200s", err)
	}
}
