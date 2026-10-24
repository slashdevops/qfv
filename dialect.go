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
// A Dialect is one of the values below, each a version of its SQL: PostgreSQL
// 18, SQL:2023, MySQL 8.4. The zero value is DialectPostgreSQL, the grammar
// this package has always parsed.
type Dialect struct {
	name string
}

var (
	// DialectPostgreSQL is PostgreSQL's grammar, as PostgreSQL 18 runs it
	// (https://www.postgresql.org/docs/18/sql-syntax.html): every operator
	// this package knows. The default.
	DialectPostgreSQL = Dialect{}

	// DialectSQLStandard is the SQL standard, ISO/IEC 9075:2023 (SQL:2023),
	// Part 2 (Foundation). It is PostgreSQL 18's grammar without what
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

	// DialectMySQL is MySQL's grammar, as MySQL 8.4 runs it with its default
	// sql_mode (https://dev.mysql.com/doc/refman/8.4/en/sql-statements.html).
	// It is PostgreSQL's grammar without what MySQL 8.4 does not run:
	//
	//   - ILIKE and NOT ILIKE, and the ~~, ~~*, !~~ and !~~* operators;
	//   - SIMILAR TO, and the POSIX operators ~, ~*, !~ and !~* (MySQL's
	//     regular expressions are REGEXP, which this package does not parse);
	//   - IS [NOT] DISTINCT FROM (MySQL's null-safe equality is <=>, which
	//     this package does not parse);
	//   - BETWEEN SYMMETRIC and BETWEEN ASYMMETRIC;
	//   - the ISNULL and NOTNULL shorthands (MySQL's ISNULL is a function).
	//
	// It keeps != and an empty ESCAPE, which MySQL 8.4 runs.
	//
	// And it reads a string as MySQL does: a backslash escapes the character
	// after it, so 'a\'b' is one string, a'b. PostgreSQL's strings keep a
	// backslash as it is, and read the same text as the string 'a\' followed
	// by b'. A filter checked with PostgreSQL's rules can hide a predicate
	// from the field allow-list when MySQL runs it: name = '\'' OR id > 0 #'
	// is one comparison of name with a string to PostgreSQL's reading, and to
	// MySQL 8.4 a comparison of name with a quote, then id > 0, then a
	// comment to the end of the line. With this dialect it is refused: id is
	// not allowed, and # is not part of the grammar.
	//
	// MySQL's sql_mode can turn backslash escapes off (NO_BACKSLASH_ESCAPES).
	// The parser cannot see the server's mode: this dialect is MySQL 8.4's
	// default, which has them on.
	DialectMySQL = Dialect{name: "MySQL 8.4"}
)

// String is the dialect's name with its version, as error messages write it:
// "PostgreSQL 18", "SQL:2023", "MySQL 8.4".
func (d Dialect) String() string {
	if d.name == "" {
		return "PostgreSQL 18"
	}

	return d.name
}

// standard reports whether the dialect is the SQL standard.
func (d Dialect) standard() bool {
	return d == DialectSQLStandard
}

// backslashEscapes reports whether a backslash in a string escapes the
// character after it, as it does in MySQL 8.4's default sql_mode.
func (d Dialect) backslashEscapes() bool {
	return d == DialectMySQL
}

// hasSpelling reports whether the dialect has this spelling of an operator it
// has. PostgreSQL 18 has every one.
func (d Dialect) hasSpelling(spelling string) bool {
	switch d {
	case DialectSQLStandard:
		switch spelling {
		case "!=", "~~", "!~~", "ISNULL", "NOTNULL":
			return false
		}
	case DialectMySQL:
		switch spelling {
		case "~~", "!~~", "ISNULL", "NOTNULL", "BETWEEN SYMMETRIC", "BETWEEN ASYMMETRIC":
			return false
		}
	}

	return true
}

// emptyEscape reports whether the dialect takes an empty ESCAPE string, which
// turns escaping off. PostgreSQL 18 and MySQL 8.4 do; the SQL standard does not.
func (d Dialect) emptyEscape() bool {
	return !d.standard()
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
	switch d {
	case DialectSQLStandard:
		return op != OpILike && op != OpRegexMatch
	case DialectMySQL:
		return op != OpILike && op != OpSimilarTo && op != OpRegexMatch && op != OpIsDistinctFrom
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
