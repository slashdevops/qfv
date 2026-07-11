package qfv

import (
	"reflect"
	"testing"
	"text/scanner"
)

func lit(s string) *LiteralNode {
	return &LiteralNode{Value: s, Kind: reflect.String, Text: "'" + s + "'"}
}

func ident(name string) *IdentifierNode { return &IdentifierNode{Name: name} }

// TestNodeStringRepresentations covers String()/Type()/Pos() for every node,
// including the negated and case-insensitive variants.
func TestNodeStringRepresentations(t *testing.T) {
	pos := scanner.Position{Filename: "t", Line: 2, Column: 3}

	tests := []struct {
		name     string
		node     Node
		wantType NodeType
		wantStr  string
	}{
		{
			name:     "UnaryOperator NOT",
			node:     &UnaryOperatorNode{baseNode: baseNode{pos: pos}, Operator: TokenOperatorNot, X: lit("x")},
			wantType: NodeTypeUnaryOperator,
			wantStr:  "(NOT 'x')",
		},
		{
			name:     "BinaryOperator AND",
			node:     &BinaryOperatorNode{baseNode: baseNode{pos: pos}, Left: ident("a"), Right: lit("x"), Operator: TokenOperatorAnd},
			wantType: NodeTypeBinaryOperator,
			wantStr:  "(a AND 'x')",
		},
		{
			name:     "Group",
			node:     &GroupNode{baseNode: baseNode{pos: pos}, Expression: ident("a")},
			wantType: NodeTypeGroup,
			wantStr:  "(a)",
		},
		{
			name:     "IsNull IS NOT NULL",
			node:     &IsNullNode{baseNode: baseNode{pos: pos}, Field: ident("a"), IsNot: true},
			wantType: NodeTypeIsNotNull,
			wantStr:  "a IS NOT NULL",
		},
		{
			name:     "In NOT IN",
			node:     &InNode{baseNode: baseNode{pos: pos}, Field: ident("a"), IsNot: true, Values: []Node{lit("x"), lit("y")}},
			wantType: NodeTypeNotIn,
			wantStr:  "a NOT IN ('x', 'y')",
		},
		{
			name:     "Between NOT SYMMETRIC",
			node:     &BetweenNode{baseNode: baseNode{pos: pos}, Field: ident("age"), Lower: lit("1"), Upper: lit("9"), IsNot: true, IsSymmetric: true},
			wantType: NodeTypeNotBetween,
			wantStr:  "age NOT BETWEEN SYMMETRIC '1' AND '9'",
		},
		{
			name:     "SimilarTo NOT",
			node:     &SimilarToNode{baseNode: baseNode{pos: pos}, Field: ident("a"), Pattern: lit("p"), IsNot: true},
			wantType: NodeTypeNotSimilarTo,
			wantStr:  "a NOT SIMILAR TO 'p'",
		},
		{
			name:     "SimilarTo plain",
			node:     &SimilarToNode{baseNode: baseNode{pos: pos}, Field: ident("a"), Pattern: lit("p")},
			wantType: NodeTypeSimilarTo,
			wantStr:  "a SIMILAR TO 'p'",
		},
		{
			name:     "Distinct without value",
			node:     &DistinctNode{baseNode: baseNode{pos: pos}, Field: ident("a")},
			wantType: NodeTypeDistinct,
			wantStr:  "a IS DISTINCT",
		},
		{
			name:     "BooleanTest IS FALSE",
			node:     &BooleanTestNode{baseNode: baseNode{pos: pos}, Field: ident("ok"), Value: BooleanFalse},
			wantType: NodeTypeBooleanTest,
			wantStr:  "ok IS FALSE",
		},
		{
			name:     "RegexMatch CS",
			node:     &RegexMatchNode{baseNode: baseNode{pos: pos}, Field: ident("a"), Pattern: lit("p")},
			wantType: NodeTypeRegexMatch,
			wantStr:  "a ~ 'p'",
		},
		{
			name:     "RegexMatch NOT CI",
			node:     &RegexMatchNode{baseNode: baseNode{pos: pos}, Field: ident("a"), Pattern: lit("p"), IsNot: true, IsCaseInsensitive: true},
			wantType: NodeTypeRegexMatch,
			wantStr:  "a !~* 'p'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.Type(); got != tt.wantType {
				t.Errorf("Type() = %s, want %s", got, tt.wantType)
			}
			if got := tt.node.String(); got != tt.wantStr {
				t.Errorf("String() = %q, want %q", got, tt.wantStr)
			}
			if got := tt.node.Pos(); got != pos {
				t.Errorf("Pos() = %v, want %v", got, pos)
			}
		})
	}
}

// TestLexerCursorHelpers covers Peek/Next/Backup/Current.
func TestLexerCursorHelpers(t *testing.T) {
	l := NewLexer("a = 'x'")
	l.Parse()

	if got := l.Peek().Value; got != "a" {
		t.Errorf("Peek() before consuming = %q, want a", got)
	}
	first := l.Next()
	if first.Value != "a" {
		t.Errorf("Next() = %q, want a", first.Value)
	}
	if got := l.Current().Value; got != "a" {
		t.Errorf("Current() = %q, want a", got)
	}
	// Peek should show the next token without consuming.
	if got := l.Peek().Type; got != TokenOperatorEqual {
		t.Errorf("Peek() = %s, want =", got)
	}
	l.Next() // '='
	l.Backup()
	if got := l.Next().Type; got != TokenOperatorEqual {
		t.Errorf("after Backup, Next() = %s, want =", got)
	}
}

// TestParsePrimaryLiteralKinds ensures int, float and boolean literals parse.
func TestParsePrimaryLiteralKinds(t *testing.T) {
	p := NewFilterParser([]string{"age", "score", "active"})

	cases := map[string]reflect.Kind{
		"age = 30":       reflect.Int64,
		"score = 3.14":   reflect.Float64,
		"active = true":  reflect.Bool,
		"active = false": reflect.Bool,
		"active = YES":   reflect.Bool,
	}
	for in, wantKind := range cases {
		t.Run(in, func(t *testing.T) {
			node, err := p.Parse(in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", in, err)
			}
			bin, ok := node.(*BinaryOperatorNode)
			if !ok {
				t.Fatalf("expected *BinaryOperatorNode, got %T", node)
			}
			rhs, ok := bin.Right.(*LiteralNode)
			if !ok {
				t.Fatalf("expected *LiteralNode rhs, got %T", bin.Right)
			}
			if rhs.Kind != wantKind {
				t.Errorf("Kind = %v, want %v", rhs.Kind, wantKind)
			}
		})
	}
}
