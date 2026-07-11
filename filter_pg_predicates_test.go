package qfv

import "testing"

// TestFilterParser_PostgreSQLPredicates exercises the PostgreSQL 18 comparison
// and pattern-matching predicates added to the grammar.
func TestFilterParser_PostgreSQLPredicates(t *testing.T) {
	allowed := []string{"name", "email", "age", "active", "status"}

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkNode func(t *testing.T, node Node)
	}{
		// ---- ILIKE / NOT ILIKE ----
		{
			name:  "ILIKE keyword",
			input: "name ILIKE 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				b := mustBinary(t, node)
				if b.Operator != TokenOperatorILike {
					t.Errorf("operator = %s, want ILIKE", b.Operator)
				}
			},
		},
		{
			name:  "NOT ILIKE keyword",
			input: "name NOT ILIKE 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				if got := mustBinary(t, node).Operator; got != TokenOperatorNotILike {
					t.Errorf("operator = %s, want NOT ILIKE", got)
				}
			},
		},
		{
			name:  "~~ operator is LIKE",
			input: "name ~~ 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				if got := mustBinary(t, node).Operator; got != TokenOperatorLike {
					t.Errorf("operator = %s, want LIKE", got)
				}
			},
		},
		{
			name:  "~~* operator is ILIKE",
			input: "name ~~* 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				if got := mustBinary(t, node).Operator; got != TokenOperatorILike {
					t.Errorf("operator = %s, want ILIKE", got)
				}
			},
		},
		{
			name:  "!~~ operator is NOT LIKE",
			input: "name !~~ 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				if got := mustBinary(t, node).Operator; got != TokenOperatorNotLike {
					t.Errorf("operator = %s, want NOT LIKE", got)
				}
			},
		},
		{
			name:  "!~~* operator is NOT ILIKE",
			input: "name !~~* 'jo%'",
			checkNode: func(t *testing.T, node Node) {
				if got := mustBinary(t, node).Operator; got != TokenOperatorNotILike {
					t.Errorf("operator = %s, want NOT ILIKE", got)
				}
			},
		},

		// ---- IS [NOT] DISTINCT FROM ----
		{
			name:  "IS DISTINCT FROM",
			input: "age IS DISTINCT FROM 30",
			checkNode: func(t *testing.T, node Node) {
				d, ok := node.(*DistinctNode)
				if !ok {
					t.Fatalf("expected *DistinctNode, got %T", node)
				}
				if d.IsNot {
					t.Error("expected IsNot=false")
				}
				if d.Value == nil {
					t.Error("expected value preserved")
				}
				if got, want := d.String(), "age IS DISTINCT FROM 30"; got != want {
					t.Errorf("String() = %q, want %q", got, want)
				}
			},
		},
		{
			name:  "IS NOT DISTINCT FROM",
			input: "age IS NOT DISTINCT FROM 30",
			checkNode: func(t *testing.T, node Node) {
				d, ok := node.(*DistinctNode)
				if !ok {
					t.Fatalf("expected *DistinctNode, got %T", node)
				}
				if !d.IsNot {
					t.Error("expected IsNot=true")
				}
				if d.Type() != NodeTypeNotDistinct {
					t.Errorf("Type() = %s, want NOT_DISTINCT", d.Type())
				}
			},
		},

		// ---- IS [NOT] TRUE / FALSE / UNKNOWN ----
		{
			name:  "IS TRUE",
			input: "active IS TRUE",
			checkNode: func(t *testing.T, node Node) {
				checkBoolTest(t, node, BooleanTrue, false)
			},
		},
		{
			name:  "IS NOT TRUE",
			input: "active IS NOT TRUE",
			checkNode: func(t *testing.T, node Node) {
				checkBoolTest(t, node, BooleanTrue, true)
			},
		},
		{
			name:  "IS FALSE",
			input: "active IS FALSE",
			checkNode: func(t *testing.T, node Node) {
				checkBoolTest(t, node, BooleanFalse, false)
			},
		},
		{
			name:  "IS NOT FALSE",
			input: "active IS NOT FALSE",
			checkNode: func(t *testing.T, node Node) {
				checkBoolTest(t, node, BooleanFalse, true)
			},
		},
		{
			name:  "IS UNKNOWN",
			input: "active IS UNKNOWN",
			checkNode: func(t *testing.T, node Node) {
				checkBoolTest(t, node, BooleanUnknown, false)
			},
		},
		{
			name:  "IS NOT UNKNOWN",
			input: "active IS NOT UNKNOWN",
			checkNode: func(t *testing.T, node Node) {
				n := checkBoolTest(t, node, BooleanUnknown, true)
				if got, want := n.String(), "active IS NOT UNKNOWN"; got != want {
					t.Errorf("String() = %q, want %q", got, want)
				}
			},
		},

		// ---- BETWEEN SYMMETRIC ----
		{
			name:  "BETWEEN SYMMETRIC",
			input: "age BETWEEN SYMMETRIC 30 AND 10",
			checkNode: func(t *testing.T, node Node) {
				b, ok := node.(*BetweenNode)
				if !ok {
					t.Fatalf("expected *BetweenNode, got %T", node)
				}
				if !b.IsSymmetric {
					t.Error("expected IsSymmetric=true")
				}
				if b.IsNot {
					t.Error("expected IsNot=false")
				}
			},
		},
		{
			name:  "NOT BETWEEN SYMMETRIC",
			input: "age NOT BETWEEN SYMMETRIC 30 AND 10",
			checkNode: func(t *testing.T, node Node) {
				b, ok := node.(*BetweenNode)
				if !ok {
					t.Fatalf("expected *BetweenNode, got %T", node)
				}
				if !b.IsSymmetric || !b.IsNot {
					t.Errorf("expected IsSymmetric=true IsNot=true, got %v %v", b.IsSymmetric, b.IsNot)
				}
				if got, want := b.String(), "age NOT BETWEEN SYMMETRIC 30 AND 10"; got != want {
					t.Errorf("String() = %q, want %q", got, want)
				}
			},
		},
		{
			name:  "BETWEEN ASYMMETRIC is default",
			input: "age BETWEEN ASYMMETRIC 10 AND 30",
			checkNode: func(t *testing.T, node Node) {
				b, ok := node.(*BetweenNode)
				if !ok {
					t.Fatalf("expected *BetweenNode, got %T", node)
				}
				if b.IsSymmetric {
					t.Error("expected IsSymmetric=false for ASYMMETRIC")
				}
			},
		},

		// ---- ISNULL / NOTNULL shorthands ----
		{
			name:  "ISNULL shorthand",
			input: "email ISNULL",
			checkNode: func(t *testing.T, node Node) {
				n, ok := node.(*IsNullNode)
				if !ok {
					t.Fatalf("expected *IsNullNode, got %T", node)
				}
				if n.IsNot {
					t.Error("expected IsNot=false")
				}
			},
		},
		{
			name:  "NOTNULL shorthand",
			input: "email NOTNULL",
			checkNode: func(t *testing.T, node Node) {
				n, ok := node.(*IsNullNode)
				if !ok {
					t.Fatalf("expected *IsNullNode, got %T", node)
				}
				if !n.IsNot {
					t.Error("expected IsNot=true")
				}
				if got, want := n.String(), "email IS NOT NULL"; got != want {
					t.Errorf("String() = %q, want %q", got, want)
				}
			},
		},

		// ---- error cases ----
		{name: "IS with typo", input: "active IS TRU", wantErr: true},
		{name: "IS DISTINCT without FROM", input: "age IS DISTINCT 30", wantErr: true},
		{name: "ILIKE without pattern", input: "name ILIKE", wantErr: true},
		{name: "dangling IS", input: "active IS", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFilterParser(allowed)
			node, err := p.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tt.checkNode != nil {
				tt.checkNode(t, node)
			}
		})
	}
}

func mustBinary(t *testing.T, node Node) *BinaryOperatorNode {
	t.Helper()
	b, ok := node.(*BinaryOperatorNode)
	if !ok {
		t.Fatalf("expected *BinaryOperatorNode, got %T", node)
	}
	return b
}

func checkBoolTest(t *testing.T, node Node, want BooleanTruthValue, isNot bool) *BooleanTestNode {
	t.Helper()
	n, ok := node.(*BooleanTestNode)
	if !ok {
		t.Fatalf("expected *BooleanTestNode, got %T", node)
	}
	if n.Value != want {
		t.Errorf("Value = %s, want %s", n.Value, want)
	}
	if n.IsNot != isNot {
		t.Errorf("IsNot = %v, want %v", n.IsNot, isNot)
	}
	if n.Type() != NodeTypeBooleanTest {
		t.Errorf("Type() = %s, want BOOLEAN_TEST", n.Type())
	}
	if n.Pos().Line == 0 && n.Pos().Column == 0 {
		// position should be populated from the IS token
		t.Log("note: boolean test node has zero position")
	}
	return n
}
