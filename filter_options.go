package qfv

// Operator identifies a filter verb that can be allowed or disallowed on a
// FilterParser. Values are human-readable and used verbatim in error messages.
type Operator string

const (
	// Logical operators.
	OpAnd Operator = "AND" // a AND b
	OpOr  Operator = "OR"  // a OR b
	OpNot Operator = "NOT" // NOT (a) — the standalone unary NOT only

	// Comparison operators.
	OpEqual              Operator = "="  // = (also drives the != / <> aliases)
	OpNotEqual           Operator = "<>" // <> and !=
	OpLessThan           Operator = "<"
	OpLessThanOrEqual    Operator = "<="
	OpGreaterThan        Operator = ">"
	OpGreaterThanOrEqual Operator = ">="

	// Pattern-matching operators. Each verb covers its negated form too, e.g.
	// OpLike governs both LIKE and NOT LIKE; OpRegexMatch governs ~ ~* !~ !~*.
	OpLike       Operator = "LIKE"       // LIKE, NOT LIKE, ~~, !~~
	OpILike      Operator = "ILIKE"      // ILIKE, NOT ILIKE, ~~*, !~~*
	OpSimilarTo  Operator = "SIMILAR TO" // SIMILAR TO, NOT SIMILAR TO
	OpRegexMatch Operator = "~"          // ~, ~*, !~, !~*

	// Set membership and range. OpIn covers NOT IN; OpBetween covers
	// NOT BETWEEN and the SYMMETRIC/ASYMMETRIC modifiers.
	OpIn      Operator = "IN"
	OpBetween Operator = "BETWEEN"

	// IS-family predicates. OpIsNull covers IS NULL, IS NOT NULL and the
	// ISNULL/NOTNULL shorthands; OpBooleanTest covers IS [NOT] TRUE/FALSE/UNKNOWN;
	// OpIsDistinctFrom covers IS [NOT] DISTINCT FROM.
	OpIsNull         Operator = "IS NULL"
	OpBooleanTest    Operator = "IS BOOLEAN"
	OpIsDistinctFrom Operator = "IS DISTINCT FROM"
)

// OperatorGroup is a convenience preset that expands to a set of Operators.
type OperatorGroup string

const (
	GroupLogical    OperatorGroup = "LOGICAL"    // AND, OR, NOT
	GroupComparison OperatorGroup = "COMPARISON" // = <> < <= > >=
	GroupPattern    OperatorGroup = "PATTERN"    // LIKE, ILIKE, SIMILAR TO, regex
	GroupMembership OperatorGroup = "MEMBERSHIP" // IN
	GroupRange      OperatorGroup = "RANGE"      // BETWEEN
	GroupNull       OperatorGroup = "NULL"       // IS NULL / ISNULL
	GroupBoolean    OperatorGroup = "BOOLEAN"    // IS TRUE/FALSE/UNKNOWN
	GroupDistinct   OperatorGroup = "DISTINCT"   // IS DISTINCT FROM
)

var operatorGroups = map[OperatorGroup][]Operator{
	GroupLogical:    {OpAnd, OpOr, OpNot},
	GroupComparison: {OpEqual, OpNotEqual, OpLessThan, OpLessThanOrEqual, OpGreaterThan, OpGreaterThanOrEqual},
	GroupPattern:    {OpLike, OpILike, OpSimilarTo, OpRegexMatch},
	GroupMembership: {OpIn},
	GroupRange:      {OpBetween},
	GroupNull:       {OpIsNull},
	GroupBoolean:    {OpBooleanTest},
	GroupDistinct:   {OpIsDistinctFrom},
}

// Operators returns the operators a group expands to. The returned slice is a
// copy and safe to modify.
func (g OperatorGroup) Operators() []Operator {
	ops := operatorGroups[g]
	out := make([]Operator, len(ops))
	copy(out, ops)
	return out
}

// AllOperators returns every operator the filter parser understands. This is the
// implicit default when no WithAllowedOperators/WithAllowedOperatorGroups option
// is supplied.
func AllOperators() []Operator {
	return []Operator{
		OpAnd, OpOr, OpNot,
		OpEqual, OpNotEqual, OpLessThan, OpLessThanOrEqual, OpGreaterThan, OpGreaterThanOrEqual,
		OpLike, OpILike, OpSimilarTo, OpRegexMatch,
		OpIn, OpBetween, OpIsNull, OpBooleanTest, OpIsDistinctFrom,
	}
}

// FilterOption configures a FilterParser at construction time.
type FilterOption func(*FilterParser)

// WithAllowedOperators restricts the filter parser to the given operators.
// It is additive with WithAllowedOperatorGroups. When neither option is used,
// all operators are allowed.
func WithAllowedOperators(ops ...Operator) FilterOption {
	return func(p *FilterParser) {
		if p.allowedOps == nil {
			p.allowedOps = make(map[Operator]struct{})
		}
		for _, o := range ops {
			p.allowedOps[o] = struct{}{}
		}
	}
}

// WithAllowedOperatorGroups restricts the filter parser to the operators in the
// given groups. It is additive with WithAllowedOperators. When neither option is
// used, all operators are allowed.
func WithAllowedOperatorGroups(groups ...OperatorGroup) FilterOption {
	return func(p *FilterParser) {
		if p.allowedOps == nil {
			p.allowedOps = make(map[Operator]struct{})
		}
		for _, g := range groups {
			for _, o := range operatorGroups[g] {
				p.allowedOps[o] = struct{}{}
			}
		}
	}
}

// comparisonOperator maps a comparison token to its Operator.
func comparisonOperator(tok TokenType) Operator {
	switch tok {
	case TokenOperatorEqual:
		return OpEqual
	case TokenOperatorNotEqual, TokenOperatorNotEqualAlias:
		return OpNotEqual
	case TokenOperatorLessThan:
		return OpLessThan
	case TokenOperatorLessThanOrEqualTo:
		return OpLessThanOrEqual
	case TokenOperatorGreaterThan:
		return OpGreaterThan
	case TokenOperatorGreaterThanOrEqualTo:
		return OpGreaterThanOrEqual
	}
	return ""
}
