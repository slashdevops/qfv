// Package qfv parses and validates the query expressions of REST APIs:
// fields selection, filtering and sorting, each gated by an allow-list of
// fields.
//
// The filter parser reads an SQL boolean expression and returns its AST. It
// checks an expression and does not translate one, so what it accepts is
// what the SQL it will run as accepts: by default PostgreSQL 18's grammar, as
// PostgreSQL 18 runs it; with WithDialect(DialectSQLStandard), the SQL
// standard's (ISO/IEC 9075:2023), which refuses PostgreSQL's own spellings
// such as != and ILIKE. WithAllowedOperators and WithAllowedOperatorGroups
// narrow the operators further. docs/dialects.md explains the dialects.
//
// Every parser is safe for concurrent use. The documentation, with the full
// grammar, is in the repository's docs directory.
package qfv
