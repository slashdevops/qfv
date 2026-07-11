package qfv

import (
	"sync"
	"testing"
)

// TestFilterParser_RejectsTrailingTokens ensures the parser no longer silently
// accepts input where only a prefix forms a valid expression.
func TestFilterParser_RejectsTrailingTokens(t *testing.T) {
	allowed := []string{"first_name", "age"}
	cases := []string{
		"first_name = 'John' garbage trailing",
		"age > 30 40",
		"first_name = 'John' 'Doe'",
		"first_name = 'John')",
		"1 = 1", // literal on the left drops the operator; must not be accepted
	}

	p := NewFilterParser(allowed)
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := p.Parse(in); err == nil {
				t.Errorf("Parse(%q) = nil error, want error for trailing/invalid tokens", in)
			}
		})
	}
}

// TestFilterParser_DistinctPreservesValue ensures DISTINCT FROM keeps the
// compared value and records negation on the node.
func TestFilterParser_DistinctPreservesValue(t *testing.T) {
	p := NewFilterParser([]string{"name"})

	node, err := p.Parse("name IS NOT NULL AND name NOT DISTINCT FROM 'John'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = node // structure is covered in filter_parser_test.go; here we just want no error

	n, err := p.Parse("name DISTINCT FROM 'John'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := n.(*DistinctNode)
	if !ok {
		t.Fatalf("expected *DistinctNode, got %T", n)
	}
	if d.IsNot {
		t.Errorf("expected IsNot=false")
	}
	if d.Value == nil {
		t.Fatal("expected DISTINCT FROM value to be preserved, got nil")
	}
	if got, want := d.String(), "name DISTINCT FROM 'John'"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestFilterParser_ConcurrentReuse ensures a single parser instance is safe for
// concurrent use. Run with -race to catch data races.
func TestFilterParser_ConcurrentReuse(t *testing.T) {
	p := NewFilterParser([]string{"a", "b"})
	inputs := []string{
		"a = 'x' AND b = 'y'",
		"a IN ('x', 'y') OR b NOT BETWEEN 1 AND 5",
		"NOT (a = 'z') AND b IS NOT NULL",
	}

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for j := range 200 {
				if _, err := p.Parse(inputs[(seed+j)%len(inputs)]); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestLexer_PlusIsIllegal ensures a stray '+' is flagged as an illegal token
// instead of producing a token with an empty type.
func TestLexer_PlusIsIllegal(t *testing.T) {
	l := NewLexer("a + b")
	l.Parse()

	found := false
	for _, tok := range l.tokens {
		if tok.Value == "+" {
			found = true
			if tok.Type != TokenIllegal {
				t.Errorf("'+' token type = %q, want %q", tok.Type, TokenIllegal)
			}
		}
	}
	if !found {
		t.Fatal("'+' token not produced by lexer")
	}

	if _, err := NewFilterParser([]string{"a", "b"}).Parse("a + b"); err == nil {
		t.Error("Parse(\"a + b\") = nil error, want error for illegal '+' token")
	}
}
