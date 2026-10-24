package qfv

// Dialect is the SQL a filter is written in: what the parser accepts is what
// that database, or that standard, runs.
//
// The parser checks an expression; it does not translate one. A caller that
// hands the filter's text to its database, as the services built on this
// package do, needs "accepted" to mean "the database runs it", so a filter is
// validated in the grammar of the SQL it will run as. Each dialect's rules are
// checked against its reference: PostgreSQL's against PostgreSQL 18 and its
// manual's SQL Syntax chapter, the
// standard's against PostgreSQL's SQL Conformance appendix, which names the
// feature each predicate is
// (https://www.postgresql.org/docs/current/features-sql-standard.html), and
// against what its manual calls an extension.
//
// A Dialect is one of the values below; the zero value is DialectPostgreSQL,
// the grammar this package has always parsed.
type Dialect struct {
	name string
}

var (
	// DialectPostgreSQL is PostgreSQL's grammar, as PostgreSQL 18 runs it
	// (https://www.postgresql.org/docs/18/sql-syntax.html): every operator
	// this package knows. The default.
	DialectPostgreSQL = Dialect{}

	// DialectSQLStandard is the SQL standard, ISO/IEC 9075:2023 (SQL:2023),
	// Part 2 (Foundation). It is PostgreSQL's grammar without what
	// PostgreSQL's manual calls its own:
	//
	//   - != (the standard's "not equal" is <>);
	//   - ILIKE and NOT ILIKE, and the ~~, ~~*, !~~ and !~~* operators;
	//   - the POSIX regular-expression operators ~, ~*, !~ and !~* (the
	//     standard's is LIKE_REGEX, which this package does not parse);
	//   - the ISNULL and NOTNULL shorthands (the standard writes IS [NOT] NULL);
	//   - an empty ESCAPE (the standard's escape character is one character).
	//
	// What it keeps is in the standard: the comparison predicate (E061-01),
	// BETWEEN (E061-02) and its SYMMETRIC form (T461), IN with a list of
	// values (E061-03), LIKE (E061-04) and its ESCAPE clause (E061-05), the
	// NULL predicate (E061-06), SIMILAR TO (T141), IS [NOT] DISTINCT FROM
	// (T151, T152), the truth value tests IS [NOT] TRUE/FALSE/UNKNOWN (F571),
	// and AND, OR and NOT.
	//
	// One difference cannot be checked: without an ESCAPE clause the standard
	// has no escape character, where PostgreSQL's is the backslash. A pattern
	// with a backslash in it means one thing to each.
	DialectSQLStandard = Dialect{name: "SQL:2023"}
)

// String is the dialect's name, as error messages write it.
func (d Dialect) String() string {
	if d.name == "" {
		return "PostgreSQL"
	}

	return d.name
}

// standard reports whether the dialect is the SQL standard.
func (d Dialect) standard() bool {
	return d == DialectSQLStandard
}

// Operators returns the operators the dialect has. The returned slice is a
// copy and safe to modify.
func (d Dialect) Operators() []Operator {
	var out []Operator

	for _, op := range AllOperators() {
		if d.has(op) {
			out = append(out, op)
		}
	}

	return out
}

// has reports whether the dialect has the operator at all. A spelling the
// dialect does not have (!= for <>, ~~ for LIKE, ISNULL for IS NULL) is
// refused where the parser reads it.
func (d Dialect) has(op Operator) bool {
	if !d.standard() {
		return true
	}

	switch op {
	case OpILike, OpRegexMatch:
		return false
	default:
		return true
	}
}

// WithDialect sets the SQL a filter is written in. Without it the dialect is
// DialectPostgreSQL. It narrows what WithAllowedOperators and
// WithAllowedOperatorGroups allow: an operator the dialect does not have is
// refused whether it is allowed or not.
func WithDialect(d Dialect) FilterOption {
	return func(p *FilterParser) {
		p.dialect = d
	}
}
