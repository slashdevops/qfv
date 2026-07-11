package qfv

import (
	"fmt"
	"reflect"
	"strings"
	"text/scanner"
)

// NodeType represents the type of AST node
type NodeType string

const (
	NodeTypeBase           NodeType = "BASE"            // Base node type
	NodeTypeLiteral        NodeType = "LITERAL"         // (value, type) -> (string, number, bool)
	NodeTypeIdentifier     NodeType = "IDENTIFIER"      // (field name) -> name, age, last_name, etc.
	NodeTypeUnaryOperator  NodeType = "UNARY_OPERATOR"  // (operator, operand) -> NOT, IS NULL, IS NOT NULL
	NodeTypeBinaryOperator NodeType = "BINARY_OPERATOR" // (operator, left, right) -> AND, OR, LIKE, IN
	NodeTypeGroup          NodeType = "GROUP"           // (expression) ->  (name = "John" AND age > 30)
	NodeTypeIsNull         NodeType = "IS_NULL"         // (field) -> name IS NULL
	NodeTypeIsNotNull      NodeType = "IS_NOT_NULL"     // (field) -> name IS NOT NULL
	NodeTypeDistinct       NodeType = "DISTINCT"        // (field) -> name DISTINCT
	NodeTypeNotDistinct    NodeType = "NOT_DISTINCT"    // (field) -> name NOT DISTINCT
	NodeTypeBetween        NodeType = "BETWEEN"         // (field, lower, upper) -> age BETWEEN 30 AND 40
	NodeTypeNotBetween     NodeType = "NOT_BETWEEN"     // (field, lower, upper) -> age NOT BETWEEN 30 AND 40
	NodeTypeIn             NodeType = "IN"              // (field, values) -> name IN ("John", "Doe")
	NodeTypeNotIn          NodeType = "NOT_IN"          // (field, values) -> name NOT IN ("John", "Doe")
	NodeTypeSimilarTo      NodeType = "SIMILAR_TO"      // (field, pattern) -> name SIMILAR TO "pattern"
	NodeTypeNotSimilarTo   NodeType = "NOT_SIMILAR_TO"  // (field, pattern) -> name NOT SIMILAR TO "pattern"
	NodeTypeRegexMatch     NodeType = "REGEX_MATCH"     // (field, pattern, is_not, is_case_insensitive) -> name ~ 'pattern'

	NodeTypeSort      NodeType = "SORT"       // (field, direction) -> name ASC, age DESC
	NodeTypeSortField NodeType = "SORT_FIELD" // (field, direction) -> name ASC, age DESC
	NodeTypeFieldList NodeType = "FIELD_LIST" // (field, field, field) -> name, age, city
)

// SimilarToNode represents a SIMILAR TO expression (e.g., name SIMILAR TO "pattern")
type SimilarToNode struct {
	baseNode
	Field   Node
	Pattern Node
	IsNot   bool // true for NOT SIMILAR TO
}

func (n *SimilarToNode) Type() NodeType {
	if n.IsNot {
		return NodeTypeNotSimilarTo
	}
	return NodeTypeSimilarTo
}
func (n *SimilarToNode) String() string {
	if n.IsNot {
		return fmt.Sprintf("%s NOT SIMILAR TO %s", n.Field.String(), n.Pattern.String())
	}
	return fmt.Sprintf("%s SIMILAR TO %s", n.Field.String(), n.Pattern.String())
}
func (n *SimilarToNode) Pos() scanner.Position { return n.pos }

// RegexMatchNode represents a regex match expression (e.g., name ~ 'pattern')
type RegexMatchNode struct {
	baseNode
	Field             Node
	Pattern           Node
	IsNot             bool // true for !~ or !~*
	IsCaseInsensitive bool // true for ~* or !~*
}

func (n *RegexMatchNode) Type() NodeType { return NodeTypeRegexMatch }
func (n *RegexMatchNode) String() string {
	op := "~"
	if n.IsNot {
		op = "!" + op
	}
	if n.IsCaseInsensitive {
		op = op + "*"
	}
	return fmt.Sprintf("%s %s %s", n.Field.String(), op, n.Pattern.String())
}
func (n *RegexMatchNode) Pos() scanner.Position { return n.pos }

// String returns the string representation of the NodeType
func (nt NodeType) String() string {
	return string(nt)
}

// Node is the base interface for all AST nodes
type Node interface {
	Type() NodeType
	String() string
	Pos() scanner.Position
}

// Base node struct to hold position
type baseNode struct {
	pos scanner.Position
}

func (n *baseNode) Type() NodeType        { return NodeTypeGroup }
func (n *baseNode) Pos() scanner.Position { return n.pos }
func (n *baseNode) String() string        { return "" }

// LiteralNode represents a literal value (string, number, bool)
type LiteralNode struct {
	baseNode
	Value any
	Kind  reflect.Kind // Kind of the value (string, number, bool)
	Text  string       // Original text representation
}

func (n *LiteralNode) Type() NodeType { return NodeTypeLiteral }
func (n *LiteralNode) String() string {
	switch n.Kind {
	case reflect.String, reflect.Int, reflect.Int64, reflect.Float32, reflect.Float64:
		return n.Text
	case reflect.Bool:
		if n.Value.(bool) {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}
func (n *LiteralNode) Pos() scanner.Position { return n.pos }

// IdentifierNode represents a field name
type IdentifierNode struct {
	baseNode
	Name string
}

func (n *IdentifierNode) Type() NodeType        { return NodeTypeIdentifier }
func (n *IdentifierNode) String() string        { return n.Name }
func (n *IdentifierNode) Pos() scanner.Position { return n.pos }

// UnaryOperatorNode (e.g., NOT name, IS NULL name)
type UnaryOperatorNode struct {
	baseNode
	Operator TokenType // "NOT",
	X        Node
}

func (n *UnaryOperatorNode) Type() NodeType        { return NodeTypeUnaryOperator }
func (n *UnaryOperatorNode) String() string        { return fmt.Sprintf("(%s %s)", n.Operator, n.X.String()) }
func (n *UnaryOperatorNode) Pos() scanner.Position { return n.pos }

// BinaryOperatorNode represents a binary operation (AND, OR)
type BinaryOperatorNode struct {
	baseNode
	Left     Node
	Right    Node
	Operator TokenType
}

func (n *BinaryOperatorNode) Type() NodeType { return NodeTypeBinaryOperator }
func (n *BinaryOperatorNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Operator, n.Right.String())
}
func (n *BinaryOperatorNode) Pos() scanner.Position { return n.pos }

// GroupNode represents a grouped expression (e.g., (name = "John" AND age > 30))
type GroupNode struct {
	baseNode
	Expression Node
}

func (n *GroupNode) Type() NodeType        { return NodeTypeGroup }
func (n *GroupNode) String() string        { return fmt.Sprintf("(%s)", n.Expression.String()) }
func (n *GroupNode) Pos() scanner.Position { return n.pos }

// IsNullNode represents an IS NULL expression (e.g., name IS NULL)
type IsNullNode struct {
	baseNode
	Field Node
	IsNot bool // true for IS NOT NULL
}

func (n *IsNullNode) Type() NodeType {
	if n.IsNot {
		return NodeTypeIsNotNull
	}
	return NodeTypeIsNull
}
func (n *IsNullNode) String() string {
	if n.IsNot {
		return fmt.Sprintf("%s IS NOT NULL", n.Field.String())
	}
	return fmt.Sprintf("%s IS NULL", n.Field.String())
}
func (n *IsNullNode) Pos() scanner.Position { return n.pos }

// InNode represents an IN expression (e.g., name IN ("John", "Doe"))
type InNode struct {
	baseNode
	Field  Node
	IsNot  bool // true for NOT IN
	Values []Node
}

func (n *InNode) Type() NodeType {
	if n.IsNot {
		return NodeTypeNotIn
	}
	return NodeTypeIn
}
func (n *InNode) String() string {
	var values []string
	for _, v := range n.Values {
		values = append(values, v.String())
	}

	op := "IN"
	if n.IsNot {
		op = "NOT IN"
	}
	return fmt.Sprintf("%s %s (%s)", n.Field.String(), op, strings.Join(values, ", "))
}
func (n *InNode) Pos() scanner.Position { return n.pos }

// DistinctNode represents a DISTINCT FROM expression (e.g., name DISTINCT FROM 'John')
type DistinctNode struct {
	baseNode
	Field Node
	Value Node // the value being compared against; nil when absent
	IsNot bool // true for NOT DISTINCT FROM
}

func (n *DistinctNode) Type() NodeType {
	if n.IsNot {
		return NodeTypeNotDistinct
	}
	return NodeTypeDistinct
}
func (n *DistinctNode) String() string {
	op := "DISTINCT"
	if n.IsNot {
		op = "NOT DISTINCT"
	}
	if n.Value != nil {
		return fmt.Sprintf("%s %s FROM %s", n.Field.String(), op, n.Value.String())
	}
	return fmt.Sprintf("%s %s", n.Field.String(), op)
}
func (n *DistinctNode) Pos() scanner.Position { return n.pos }

// BetweenNode represents a BETWEEN expression (e.g., age BETWEEN 30 AND 40)
type BetweenNode struct {
	baseNode
	Field Node
	Lower Node
	Upper Node
	IsNot bool // true for NOT BETWEEN
}

func (n *BetweenNode) Type() NodeType {
	if n.IsNot {
		return NodeTypeNotBetween
	}
	return NodeTypeBetween
}
func (n *BetweenNode) String() string {
	op := "BETWEEN"
	if n.IsNot {
		op = "NOT BETWEEN"
	}
	return fmt.Sprintf("%s %s %s AND %s", n.Field.String(), op, n.Lower.String(), n.Upper.String())
}
func (n *BetweenNode) Pos() scanner.Position { return n.pos }
