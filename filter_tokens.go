package qfv

import (
	"fmt"
	"text/scanner"
)

// TokenType represents the type of token
type TokenType string

const (
	TokenIdentifier       TokenType = "IDENTIFIER"        // Identifier represents a variable name or keyword
	TokenOperator         TokenType = "OPERATOR"          // Operator represents an operator (e.g., =, <, >, etc.)
	TokenString           TokenType = "STRING"            // String represents a string literal
	TokenBoolean          TokenType = "BOOLEAN"           // Boolean represents a boolean literal (true/false)
	TokenLogicalOperation TokenType = "LOGICAL_OPERATION" // LogicalOperation represents a logical operation (AND, OR, NOT)
	TokenSortOperation    TokenType = "SORT_OPERATION"    // SortOperation represents a sort operation (ASC, DESC)
	TokenLPAREN           TokenType = "LPAREN"            // Parenthesis represents a parenthesis ((), ))
	TokenRPAREN           TokenType = "RPAREN"
	// ----
	TokenIllegal    TokenType = "ILLEGAL"    // Illegal represents an illegal token
	TokenEOF        TokenType = "EOF"        // EOF represents the end of the file/input
	TokenInt        TokenType = "INT"        // Int represents an integer literal
	TokenFloat      TokenType = "FLOAT"      // Float represents a floating-point literal
	TokenComma      TokenType = "COMMA"      // Comma represents a comma (,)
	TokenWhitespace TokenType = "WHITESPACE" // Whitespace represents whitespace characters (spaces, tabs, newlines)
	// ----
	TokenOperatorEqual                TokenType = "="
	TokenOperatorNotEqual             TokenType = "<>"
	TokenOperatorNotEqualAlias        TokenType = "!="
	TokenOperatorLessThan             TokenType = "<"
	TokenOperatorLessThanOrEqualTo    TokenType = "<="
	TokenOperatorGreaterThan          TokenType = ">"
	TokenOperatorGreaterThanOrEqualTo TokenType = ">="
	// ----
	TokenOperatorAnd          TokenType = "AND"
	TokenOperatorOr           TokenType = "OR"
	TokenOperatorNot          TokenType = "NOT"
	TokenOperatorLike         TokenType = "LIKE"
	TokenOperatorNotLike      TokenType = "NOT LIKE"
	TokenOperatorILike        TokenType = "ILIKE"
	TokenOperatorNotILike     TokenType = "NOT ILIKE"
	TokenOperatorIn           TokenType = "IN"
	TokenOperatorIsNotNull    TokenType = "IS NOT NULL"
	TokenOperatorIsNull       TokenType = "IS NULL"
	TokenOperatorNotIn        TokenType = "NOT IN"
	TokenOperatorBetween      TokenType = "BETWEEN"
	TokenOperatorNotBetween   TokenType = "NOT BETWEEN"
	TokenOperatorDistinct     TokenType = "DISTINCT"
	TokenOperatorNotDistinct  TokenType = "NOT DISTINCT"
	TokenOperatorSimilarTo    TokenType = "SIMILAR TO"
	TokenOperatorNotSimilarTo TokenType = "NOT SIMILAR TO"
	// ---- Regex Operators ----
	TokenOperatorRegexMatchCS    TokenType = "~"   // Case-sensitive regex match
	TokenOperatorNotRegexMatchCS TokenType = "!~"  // Case-sensitive regex non-match
	TokenOperatorRegexMatchCI    TokenType = "~*"  // Case-insensitive regex match
	TokenOperatorNotRegexMatchCI TokenType = "!~*" // Case-insensitive regex non-match
)

func (t TokenType) String() string {
	return string(t)
}

// Token represents a lexical token
type Token struct {
	Pos   scanner.Position
	Type  TokenType
	Value string // Literal value of the token
}

func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Value: %s}", t.Type, t.Value)
}
