package qfv

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"
)

type QFVFilterError struct {
	Field   string
	Message string
}

func (e *QFVFilterError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("error on field '%s': %s", e.Field, e.Message)
	}

	return fmt.Sprintf("error: %s", e.Message)
}

// FilterParser parses the query parameter for filtering.
//
// A FilterParser holds only the immutable set of allowed fields, so a single
// instance is safe for concurrent use by multiple goroutines: the mutable
// state required to parse one expression lives in a per-call filterParseState.
type FilterParser struct {
	allowedFields map[string]struct{}
}

// NewFilterParser creates a new parser with the allowed fields
func NewFilterParser(allowedFields []string) *FilterParser {
	filterFields := make(map[string]struct{}, len(allowedFields))

	for _, f := range allowedFields {
		filterFields[f] = struct{}{}
	}

	return &FilterParser{
		allowedFields: filterFields,
	}
}

// filterParseState holds the mutable state for a single Parse call. Keeping it
// out of FilterParser is what makes FilterParser safe for concurrent reuse.
type filterParseState struct {
	allowedFields map[string]struct{}
	lexer         *Lexer
	currentToken  Token
	errors        []error
}

// Parse parses the filter query and returns the AST
func (p *FilterParser) Parse(input string) (Node, error) {
	st := &filterParseState{
		allowedFields: p.allowedFields,
		lexer:         NewLexer(input),
	}
	st.lexer.Parse()

	// Check for illegal tokens in the input
	for _, token := range st.lexer.tokens {
		if token.Type == TokenIllegal {
			st.addError(&QFVFilterError{Field: token.Value, Message: "illegal token"})
		}
	}

	st.nextToken()

	if st.currentToken.Type == TokenEOF {
		return nil, &QFVFilterError{Message: "empty filter expression"}
	}

	node := st.parseExpression()

	// The whole input must be consumed. Any leftover tokens mean only a prefix
	// of the input was a valid expression (e.g. "name = 'John' garbage"), which
	// must be reported instead of silently accepted.
	if st.currentToken.Type != TokenEOF {
		st.addError(&QFVFilterError{Message: fmt.Sprintf("unexpected token %q after a complete expression", st.currentToken.Value)})
	}

	if len(st.errors) > 0 {
		return nil, errors.Join(st.errors...)
	}

	return node, nil
}

// nextToken advances to the next token
func (p *filterParseState) nextToken() {
	p.currentToken = p.lexer.Next()
}

// expect checks if the current token is of the expected type
func (p *filterParseState) expect(tokenType TokenType) bool {
	if p.currentToken.Type == tokenType {
		p.nextToken()
		return true
	}

	p.addError(&QFVFilterError{Message: fmt.Sprintf("expected %s, got %s", tokenType, p.currentToken.Type)})
	return false
}

// addError adds an error to the error list
func (p *filterParseState) addError(err error) {
	p.errors = append(p.errors, err)
}

// parseExpression parses an expression
func (p *filterParseState) parseExpression() Node {
	return p.parseLogicalOr()
}

// parseLogicalOr parses OR expressions
func (p *filterParseState) parseLogicalOr() Node {
	left := p.parseLogicalAnd()

	for p.currentToken.Type == TokenOperatorOr {
		pos := p.currentToken.Pos
		operator := p.currentToken.Type
		p.nextToken()
		right := p.parseLogicalAnd()
		left = &BinaryOperatorNode{
			baseNode: baseNode{pos: pos},
			Left:     left,
			Right:    right,
			Operator: operator,
		}
	}

	return left
}

// parseLogicalAnd parses AND expressions
func (p *filterParseState) parseLogicalAnd() Node {
	left := p.parseComparison()

	for p.currentToken.Type == TokenOperatorAnd {
		pos := p.currentToken.Pos
		operator := p.currentToken.Type
		p.nextToken()
		right := p.parseComparison()
		left = &BinaryOperatorNode{
			baseNode: baseNode{pos: pos},
			Left:     left,
			Right:    right,
			Operator: operator,
		}
	}

	return left
}

// parseComparison parses comparison expressions
func (p *filterParseState) parseComparison() Node {
	// Check for NOT operator
	if p.currentToken.Type == TokenOperatorNot {
		pos := p.currentToken.Pos
		p.nextToken()
		expr := p.parseComparison()
		return &UnaryOperatorNode{
			baseNode: baseNode{pos: pos},
			Operator: TokenOperatorNot,
			X:        expr,
		}
	}

	// Check for parenthesized expressions
	if p.currentToken.Type == TokenLPAREN {
		pos := p.currentToken.Pos
		p.nextToken()
		expr := p.parseExpression()
		if !p.expect(TokenRPAREN) {
			p.addError(&QFVFilterError{Message: "expected closing parenthesis"})
		}
		return &GroupNode{
			baseNode:   baseNode{pos: pos},
			Expression: expr,
		}
	}

	// Parse field comparison
	if p.currentToken.Type == TokenIdentifier {
		field := &IdentifierNode{
			baseNode: baseNode{pos: p.currentToken.Pos},
			Name:     p.currentToken.Value,
		}
		p.nextToken()

		// Check if field is allowed
		if _, ok := p.allowedFields[field.Name]; !ok {
			p.addError(&QFVFilterError{Field: field.Name, Message: "field not allowed"})
		}

		// Handle different operators
		switch p.currentToken.Type {
		case TokenOperatorEqual, TokenOperatorNotEqual, TokenOperatorNotEqualAlias,
			TokenOperatorLessThan, TokenOperatorLessThanOrEqualTo,
			TokenOperatorGreaterThan, TokenOperatorGreaterThanOrEqualTo:
			return p.parseComparisonOperator(field)
		case TokenOperatorLike:
			p.nextToken() // Consume LIKE
			return p.parseLikeOperator(field, false)
		case TokenOperatorIn:
			p.nextToken() // Consume IN
			return p.parseInOperator(field, false)
		case TokenOperatorBetween:
			p.nextToken() // Consume BETWEEN
			return p.parseBetweenOperator(field, false)
		case TokenOperatorIsNull:
			p.nextToken() // Consume IS
			return p.parseIsNullOperator(field)
		case TokenOperatorDistinct:
			p.nextToken() // Consume DISTINCT
			return p.parseDistinctOperator(field, false)
		case TokenOperatorSimilarTo:
			p.nextToken() // Consume SIMILAR
			return p.parseSimilarToOperator(field, false)
		case TokenOperatorRegexMatchCS, TokenOperatorNotRegexMatchCS, TokenOperatorRegexMatchCI, TokenOperatorNotRegexMatchCI:
			opToken := p.currentToken
			p.nextToken() // Consume regex operator
			patternNode := p.parsePrimary()

			// Check if the pattern is a string literal
			patternLiteral, ok := patternNode.(*LiteralNode)
			if !ok || patternLiteral.Kind != reflect.String {
				p.addError(&QFVFilterError{Message: fmt.Sprintf("expected string pattern for regex operator %s, got %s", opToken.Type, patternNode.Type())})
				// Return the field node or the invalid pattern node on error
				// Returning the pattern node might give slightly better context
				return patternNode
			}

			return &RegexMatchNode{
				baseNode:          baseNode{pos: opToken.Pos},
				Field:             field,
				Pattern:           patternNode, // Use the parsed node
				IsNot:             opToken.Type == TokenOperatorNotRegexMatchCS || opToken.Type == TokenOperatorNotRegexMatchCI,
				IsCaseInsensitive: opToken.Type == TokenOperatorRegexMatchCI || opToken.Type == TokenOperatorNotRegexMatchCI,
			}
		case TokenOperatorNot:
			// Handle NOT operators (NOT IN, NOT BETWEEN, NOT LIKE, NOT SIMILAR TO,
			// NOT DISTINCT FROM). Negation is recorded on the resulting node via its
			// IsNot flag (or the NOT LIKE operator), not by wrapping in a NOT node,
			// so that consumers get a single, self-describing node.
			p.nextToken() // Consume NOT

			switch p.currentToken.Type {
			case TokenOperatorIn:
				p.nextToken() // Consume IN
				return p.parseInOperator(field, true)
			case TokenOperatorBetween:
				p.nextToken() // Consume BETWEEN
				return p.parseBetweenOperator(field, true)
			case TokenOperatorLike:
				p.nextToken() // Consume LIKE
				return p.parseLikeOperator(field, true)
			case TokenOperatorSimilarTo:
				p.nextToken() // Consume SIMILAR
				return p.parseSimilarToOperator(field, true)
			case TokenOperatorDistinct:
				p.nextToken() // Consume DISTINCT
				return p.parseDistinctOperator(field, true)
			default:
				p.addError(&QFVFilterError{Message: fmt.Sprintf("unexpected token after NOT: %s", p.currentToken.Type)})
				return field
			}

		default:
			p.addError(&QFVFilterError{Field: field.Name, Message: "unexpected token after field"})
			return field
		}
	}

	// Parse literal
	return p.parsePrimary()
}

// parseComparisonOperator parses comparison operators (=, <>, !=, <, <=, >, >=)
func (p *filterParseState) parseComparisonOperator(field Node) Node {
	pos := p.currentToken.Pos
	operator := p.currentToken.Type
	p.nextToken()
	right := p.parsePrimary()
	return &BinaryOperatorNode{
		baseNode: baseNode{pos: pos},
		Left:     field,
		Right:    right,
		Operator: operator,
	}
}

// parseSimilarToOperator parses SIMILAR TO operator
// Expects the current token to be TO after SIMILAR was consumed.
func (p *filterParseState) parseSimilarToOperator(field Node, isNot bool) Node {
	pos := p.lexer.Current().Pos // Use position of SIMILAR token (already consumed)
	if p.currentToken.Type != TokenIdentifier || strings.ToUpper(p.currentToken.Value) != "TO" {
		p.addError(&QFVFilterError{Message: "expected TO after SIMILAR"})
		return field // Return field on error
	}
	p.nextToken() // Consume TO
	pattern := p.parsePrimary()
	return &SimilarToNode{
		baseNode: baseNode{pos: pos},
		Field:    field,
		Pattern:  pattern,
		IsNot:    isNot,
	}
}

// parseLikeOperator parses LIKE operator
// Expects the current token to be the pattern after LIKE was consumed.
func (p *filterParseState) parseLikeOperator(field Node, isNot bool) Node {
	pos := p.lexer.Current().Pos // Use position of LIKE token (already consumed)
	pattern := p.parsePrimary()
	operator := TokenOperatorLike
	if isNot {
		operator = TokenOperatorNotLike
	}
	return &BinaryOperatorNode{
		baseNode: baseNode{pos: pos},
		Left:     field,
		Right:    pattern,
		Operator: operator,
	}
}

// parseInOperator parses IN operator
// Expects the current token to be LPAREN after IN was consumed.
func (p *filterParseState) parseInOperator(field Node, isNot bool) Node {
	pos := p.lexer.Current().Pos // Use position of IN token (already consumed)
	if !p.expect(TokenLPAREN) {
		p.addError(&QFVFilterError{Message: "expected opening parenthesis after IN"})
		return field
	}

	var values []Node
	// Parse the first value
	if p.currentToken.Type == TokenRPAREN {
		p.addError(&QFVFilterError{Message: "expected at least one value after IN ("})
	} else {
		values = append(values, p.parsePrimary())
	}

	// Parse additional values
	for p.currentToken.Type == TokenComma {
		p.nextToken()
		if p.currentToken.Type == TokenRPAREN { // Handle trailing comma
			p.addError(&QFVFilterError{Message: "unexpected closing parenthesis after comma in IN list"})
			break
		}
		values = append(values, p.parsePrimary())
	}

	if !p.expect(TokenRPAREN) {
		p.addError(&QFVFilterError{Message: "expected closing parenthesis after IN values"})
	}

	return &InNode{
		baseNode: baseNode{pos: pos},
		Field:    field,
		IsNot:    isNot,
		Values:   values,
	}
}

// parseBetweenOperator parses BETWEEN operator
// Expects the current token to be the lower bound after BETWEEN was consumed.
func (p *filterParseState) parseBetweenOperator(field Node, isNot bool) Node {
	pos := p.lexer.Current().Pos // Use position of BETWEEN token (already consumed)
	lower := p.parsePrimary()

	if !p.expect(TokenOperatorAnd) {
		p.addError(&QFVFilterError{Message: "expected AND in BETWEEN expression"})
		return field
	}

	upper := p.parsePrimary()

	return &BetweenNode{
		baseNode: baseNode{pos: pos},
		Field:    field,
		Lower:    lower,
		Upper:    upper,
		IsNot:    isNot,
	}
}

// parseIsNullOperator parses IS [NOT] NULL operator
// Expects the current token to be NOT or NULL after IS was consumed.
func (p *filterParseState) parseIsNullOperator(field Node) Node {
	pos := p.lexer.Current().Pos // Use position of IS token (already consumed)
	isNot := false
	if p.currentToken.Type == TokenOperatorNot {
		isNot = true
		p.nextToken() // Consume NOT
	}

	// Check for NULL
	if p.currentToken.Type == TokenIdentifier && strings.ToUpper(p.currentToken.Value) == "NULL" {
		p.nextToken() // Consume NULL
		return &IsNullNode{
			baseNode: baseNode{pos: pos},
			Field:    field,
			IsNot:    isNot,
		}
	}

	if isNot {
		p.addError(&QFVFilterError{Message: "expected NULL after IS NOT"})
	} else {
		p.addError(&QFVFilterError{Message: "expected NULL or NOT NULL after IS"})
	}
	return field // Return field on error
}

// parseDistinctOperator parses DISTINCT FROM operator
// Expects the current token to be FROM after DISTINCT was consumed.
func (p *filterParseState) parseDistinctOperator(field Node, isNot bool) Node {
	pos := p.lexer.Current().Pos // Use position of DISTINCT token (already consumed)
	// Expect FROM (treated as identifier by lexer)
	if p.currentToken.Type != TokenIdentifier || strings.ToUpper(p.currentToken.Value) != "FROM" {
		p.addError(&QFVFilterError{Message: "expected FROM after DISTINCT"})
		return field // Return field on error
	}
	p.nextToken() // Consume FROM
	// Parse the value being compared against and keep it on the node so callers
	// can evaluate "field IS [NOT] DISTINCT FROM value".
	value := p.parsePrimary()

	return &DistinctNode{
		baseNode: baseNode{pos: pos},
		Field:    field,
		Value:    value,
		IsNot:    isNot,
	}
}

// parsePrimary parses primary expressions (literals)
func (p *filterParseState) parsePrimary() Node {
	switch p.currentToken.Type {
	case TokenString:
		node := &LiteralNode{
			baseNode: baseNode{pos: p.currentToken.Pos},
			Value:    strings.Trim(p.currentToken.Value, "'"),
			Kind:     reflect.String,
			Text:     p.currentToken.Value,
		}
		p.nextToken()
		return node

	case TokenInt:
		val, err := strconv.ParseInt(p.currentToken.Value, 10, 64)
		if err != nil {
			p.addError(&QFVFilterError{Message: fmt.Sprintf("invalid integer: %s", p.currentToken.Value)})
		}
		node := &LiteralNode{
			baseNode: baseNode{pos: p.currentToken.Pos},
			Value:    val,
			Kind:     reflect.Int64,
			Text:     p.currentToken.Value,
		}
		p.nextToken()
		return node

	case TokenFloat:
		val, err := strconv.ParseFloat(p.currentToken.Value, 64)
		if err != nil {
			p.addError(&QFVFilterError{Message: fmt.Sprintf("invalid float: %s", p.currentToken.Value)})
		}
		node := &LiteralNode{
			baseNode: baseNode{pos: p.currentToken.Pos},
			Value:    val,
			Kind:     reflect.Float64,
			Text:     p.currentToken.Value,
		}
		p.nextToken()
		return node

	case TokenBoolean:
		val := strings.ToUpper(p.currentToken.Value) == "TRUE" || strings.ToUpper(p.currentToken.Value) == "YES"
		node := &LiteralNode{
			baseNode: baseNode{pos: p.currentToken.Pos},
			Value:    val,
			Kind:     reflect.Bool,
			Text:     p.currentToken.Value,
		}
		p.nextToken()
		return node

	default:
		p.addError(&QFVFilterError{Message: fmt.Sprintf("unexpected token: %s", p.currentToken.Type)})
		// Skip the token to avoid infinite loops
		p.nextToken()
		return &LiteralNode{
			baseNode: baseNode{pos: scanner.Position{}},
			Value:    nil,
			Kind:     0,
			Text:     "",
		}
	}
}
