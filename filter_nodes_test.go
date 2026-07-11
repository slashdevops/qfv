package qfv

import (
	"reflect"
	"testing"
	"text/scanner"
)

func TestNodeType_String(t *testing.T) {
	tests := []struct {
		name     string
		nodeType NodeType
		want     string
	}{
		{"Base", NodeTypeBase, "BASE"},
		{"Literal", NodeTypeLiteral, "LITERAL"},
		{"Identifier", NodeTypeIdentifier, "IDENTIFIER"},
		{"UnaryOperator", NodeTypeUnaryOperator, "UNARY_OPERATOR"},
		{"BinaryOperator", NodeTypeBinaryOperator, "BINARY_OPERATOR"},
		{"Group", NodeTypeGroup, "GROUP"},
		{"IsNull", NodeTypeIsNull, "IS_NULL"},
		{"IsNotNull", NodeTypeIsNotNull, "IS_NOT_NULL"},
		{"Distinct", NodeTypeDistinct, "DISTINCT"},
		{"NotDistinct", NodeTypeNotDistinct, "NOT_DISTINCT"},
		{"Between", NodeTypeBetween, "BETWEEN"},
		{"NotBetween", NodeTypeNotBetween, "NOT_BETWEEN"},
		{"In", NodeTypeIn, "IN"},
		{"NotIn", NodeTypeNotIn, "NOT_IN"},
		{"Sort", NodeTypeSort, "SORT"},
		{"SortField", NodeTypeSortField, "SORT_FIELD"},
		{"FieldList", NodeTypeFieldList, "FIELD_LIST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nodeType.String(); got != tt.want {
				t.Errorf("NodeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	node := &baseNode{pos: pos}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeGroup {
			t.Errorf("baseNode.Type() = %v, want %v", got, NodeTypeGroup)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("baseNode.Pos() = %v, want %v", got, pos)
		}
	})

	t.Run("String", func(t *testing.T) {
		if got := node.String(); got != "" {
			t.Errorf("baseNode.String() = %v, want %v", got, "")
		}
	})
}

func TestLiteralNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name     string
		node     *LiteralNode
		wantType NodeType
		wantStr  string
	}{
		{
			name: "String",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    "test",
				Kind:     reflect.String,
				Text:     "'test'",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "'test'",
		},
		{
			name: "Integer",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    42,
				Kind:     reflect.Int,
				Text:     "42",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "42",
		},
		{
			name: "Float",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    3.14,
				Kind:     reflect.Float64,
				Text:     "3.14",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "3.14",
		},
		{
			name: "Boolean True",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    true,
				Kind:     reflect.Bool,
				Text:     "true",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "true",
		},
		{
			name: "Boolean False",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    false,
				Kind:     reflect.Bool,
				Text:     "false",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "false",
		},
		{
			name: "Unknown Kind",
			node: &LiteralNode{
				baseNode: baseNode{pos: pos},
				Value:    struct{}{},
				Kind:     reflect.Struct,
				Text:     "{}",
			},
			wantType: NodeTypeLiteral,
			wantStr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.Type(); got != tt.wantType {
				t.Errorf("LiteralNode.Type() = %v, want %v", got, tt.wantType)
			}
			if got := tt.node.String(); got != tt.wantStr {
				t.Errorf("LiteralNode.String() = %v, want %v", got, tt.wantStr)
			}
			if got := tt.node.Pos(); got != pos {
				t.Errorf("LiteralNode.Pos() = %v, want %v", got, pos)
			}
		})
	}
}

func TestIdentifierNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	node := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "fieldName",
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeIdentifier {
			t.Errorf("IdentifierNode.Type() = %v, want %v", got, NodeTypeIdentifier)
		}
	})

	t.Run("String", func(t *testing.T) {
		if got := node.String(); got != "fieldName" {
			t.Errorf("IdentifierNode.String() = %v, want %v", got, "fieldName")
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("IdentifierNode.Pos() = %v, want %v", got, pos)
		}
	})
}

func TestGroupNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	exprNode := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "expression",
	}

	node := &GroupNode{
		baseNode:   baseNode{pos: pos},
		Expression: exprNode,
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeGroup {
			t.Errorf("GroupNode.Type() = %v, want %v", got, NodeTypeGroup)
		}
	})

	t.Run("String", func(t *testing.T) {
		expected := "(expression)"
		if got := node.String(); got != expected {
			t.Errorf("GroupNode.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("GroupNode.Pos() = %v, want %v", got, pos)
		}
	})
}

func TestIsNullNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	fieldNode := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "fieldName",
	}

	node := &IsNullNode{
		baseNode: baseNode{pos: pos},
		Field:    fieldNode,
		IsNot:    false,
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeIsNull {
			t.Errorf("IsNullNode.Type() = %v, want %v", got, NodeTypeIsNull)
		}
	})

	t.Run("String", func(t *testing.T) {
		expected := "fieldName IS NULL"
		if got := node.String(); got != expected {
			t.Errorf("IsNullNode.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("IsNullNode.Pos() = %v, want %v", got, pos)
		}
	})
}

func TestInNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	fieldNode := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "fieldName",
	}

	value1 := &LiteralNode{
		baseNode: baseNode{pos: pos},
		Value:    "val1",
		Kind:     reflect.String,
		Text:     "'val1'",
	}

	value2 := &LiteralNode{
		baseNode: baseNode{pos: pos},
		Value:    "val2",
		Kind:     reflect.String,
		Text:     "'val2'",
	}

	node := &InNode{
		baseNode: baseNode{pos: pos},
		Field:    fieldNode,
		IsNot:    false,
		Values:   []Node{value1, value2},
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeIn {
			t.Errorf("InNode.Type() = %v, want %v", got, NodeTypeIn)
		}
	})

	t.Run("String", func(t *testing.T) {
		expected := "fieldName IN ('val1', 'val2')"
		if got := node.String(); got != expected {
			t.Errorf("InNode.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("InNode.Pos() = %v, want %v", got, pos)
		}
	})
}

func TestDistinctNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	fieldNode := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "fieldName",
	}

	node := &DistinctNode{
		baseNode: baseNode{pos: pos},
		Field:    fieldNode,
		IsNot:    false,
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeDistinct {
			t.Errorf("DistinctNode.Type() = %v, want %v", got, NodeTypeDistinct)
		}
	})

	t.Run("String", func(t *testing.T) {
		expected := "fieldName IS DISTINCT"
		if got := node.String(); got != expected {
			t.Errorf("DistinctNode.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("DistinctNode.Pos() = %v, want %v", got, pos)
		}
	})
}

func TestBetweenNode(t *testing.T) {
	pos := scanner.Position{Filename: "test.go", Line: 1, Column: 1}
	fieldNode := &IdentifierNode{
		baseNode: baseNode{pos: pos},
		Name:     "age",
	}

	lowerNode := &LiteralNode{
		baseNode: baseNode{pos: pos},
		Value:    20,
		Kind:     reflect.Int,
		Text:     "20",
	}

	upperNode := &LiteralNode{
		baseNode: baseNode{pos: pos},
		Value:    30,
		Kind:     reflect.Int,
		Text:     "30",
	}

	node := &BetweenNode{
		baseNode: baseNode{pos: pos},
		Field:    fieldNode,
		Lower:    lowerNode,
		Upper:    upperNode,
		IsNot:    false,
	}

	t.Run("Type", func(t *testing.T) {
		if got := node.Type(); got != NodeTypeBetween {
			t.Errorf("BetweenNode.Type() = %v, want %v", got, NodeTypeBetween)
		}
	})

	t.Run("String", func(t *testing.T) {
		expected := "age BETWEEN 20 AND 30"
		if got := node.String(); got != expected {
			t.Errorf("BetweenNode.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Pos", func(t *testing.T) {
		if got := node.Pos(); got != pos {
			t.Errorf("BetweenNode.Pos() = %v, want %v", got, pos)
		}
	})
}
