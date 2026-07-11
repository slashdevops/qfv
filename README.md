# Query Filters Validator (QFV)

A Go library for parsing and validating query expressions commonly used in REST APIs and database queries. This library provides robust parsing and validation for fields selection, filtering, and sorting operations.

## Overview

QFV (Query Filters Validator) is an Abstract Syntax Tree (AST) parser designed to validate and parse three common types of query expressions:

1. **Fields Selection**: Specify which fields to include in the response
2. **Filtering**: Apply conditions to filter results
3. **Sorting**: Define the order of returned results

The library ensures that only allowed fields are used in these expressions, helping to secure your API endpoints against invalid or malicious queries.

## Installation

```bash
go get github.com/slashdevops/qfv@latest
```

## Usage

### Basic Example

```go
package main

import (
  "fmt"
  "log"

  qfv "github.com/slashdevops/qfv"
)

func main() {
  // Define the allowed fields for your API
  allowedFields := []string{"first_name", "last_name", "email", "created_at", "updated_at"}

  // Create parsers with the allowed fields
  sortParser := qfv.NewSortParser(allowedFields)
  fieldsParser := qfv.NewFieldsParser(allowedFields)
  filterParser := qfv.NewFilterParser(allowedFields)

  // Example inputs from your API
  sortInput := "first_name ASC,created_at DESC"
  fieldsInput := "first_name, last_name, email"
  filterInput := "first_name = 'John' AND last_name = 'Doe'"

  // Parse and validate the inputs
  sortNode, err := sortParser.Parse(sortInput)
  if err != nil {
    log.Fatalf("Sort validation error: %v", err)
  }

  fieldsNode, err := fieldsParser.Parse(fieldsInput)
  if err != nil {
    log.Fatalf("Fields validation error: %v", err)
  }

  _, err = filterParser.Parse(filterInput)
  if err != nil {
    log.Fatalf("Filter validation error: %v", err)
  }

  // Use the parsed nodes in your application
  fmt.Println("Sort fields:")
  for _, field := range sortNode.Fields {
    fmt.Printf("  %s %s\n", field.Field, field.Direction)
  }

  fmt.Println("\nRequested fields:")
  for _, field := range fieldsNode.Fields {
    fmt.Printf("  %s\n", field)
  }
}
```

## Features

### Fields Parser

Parse and validate field selection expressions like:

```text
first_name, last_name, email
```

The parser ensures that:

- Only allowed fields are requested
- No empty field names are provided
- The syntax is valid

### Sort Parser

Parse and validate sort expressions like:

```sql
first_name ASC, created_at DESC
```

The parser ensures that:

- Only allowed fields are used for sorting
- Each field has a valid direction (ASC or DESC)
- The syntax is valid

### Filter Parser

Parse and validate complex filter expressions like:

```sql
first_name = 'John' AND last_name = 'Doe' OR email = 'example@example.com'
```

The filter parser supports:

- **Logical operators**: AND, OR, NOT
- **Comparison operators**: =, <>, !=, <, <=, >, >=
- **Special operators**: LIKE, IN, BETWEEN, IS NULL, IS NOT NULL, DISTINCT, SIMILAR TO
- **Regex operators**:
  - `~`: Case-sensitive regex match
  - `!~`: Case-sensitive regex non-match
  - `~*`: Case-insensitive regex match
  - `!~*`: Case-insensitive regex non-match
- **Grouping** with parentheses
- **Literals**: strings, integers, floats, booleans

The parser ensures that:

- Only allowed fields are used in filter conditions
- The syntax is valid
- The expression forms a valid abstract syntax tree (AST)

## Advanced Filter Examples

```go
// Simple comparison
"first_name = 'John'"

// Logical operators
"first_name = 'John' AND last_name = 'Doe'"
"first_name = 'John' OR first_name = 'Jane'"
"NOT (first_name = 'John')"

// Comparison operators
"age > 30"
"age >= 30"
"age < 30"
"age <= 30"
"age <> 30"  // Not equal
"age != 30"  // Not equal (alternative)

// Special operators
"first_name LIKE 'J%'"
"status IN ('active', 'pending')"
"age BETWEEN 20 AND 30"
"middle_name IS NULL"
"middle_name IS NOT NULL"
"name SIMILAR TO 'J%n'" // SQL standard regex
"name NOT SIMILAR TO 'J%n'"
"email ~ '^[^@]+@[^@]+\.[^@]+$'" // Case-sensitive regex match
"email !~ '^[^@]+@[^@]+\.[^@]+$'" // Case-sensitive regex non-match
"email ~* '(?i)^admin@'" // Case-insensitive regex match (using Go regex flag)
"email !~* '(?i)^admin@'" // Case-insensitive regex non-match (using Go regex flag)

// Complex expressions
"(first_name = 'John' OR first_name = 'Jane') AND age > 30"
"status IN ('active', 'pending') AND created_at > '2023-01-01'"
```

## Error Handling

The library provides detailed error messages for invalid expressions:

```go
_, err := filterParser.Parse("unknown_field = 'value'")
if err != nil {
    // err contains: "field unknown_field is not allowed"
}

_, err = filterParser.Parse("first_name = ")
if err != nil {
    // err contains syntax error details
}
```

The filter parser aggregates every problem it finds in a single pass using
[`errors.Join`](https://pkg.go.dev/errors#Join), so a returned error may wrap
several `*QFVFilterError` values. Use `errors.As` to inspect them:

```go
_, err := filterParser.Parse("unknown = 'x' garbage")
var fe *qfv.QFVFilterError
if errors.As(err, &fe) {
    // fe.Field / fe.Message for the first matching cause
}
```

## Breaking Changes & Migration (v0.0.x → v0.1.0)

`v0.1.0` fixes correctness bugs in the filter parser and validator. If you only
call `Parse` and check the returned `error`, most changes are transparent — the
parser now rejects inputs it previously accepted by mistake. If you **inspect the
returned AST**, review the notes below.

### 1. Trailing/incomplete input is now rejected (behavior change)

Previously the parser stopped at the first complete expression and silently
ignored the rest, so invalid input validated successfully. It now requires the
**entire** input to be consumed.

| Input | Before `v0.1.0` | `v0.1.0` |
| --- | --- | --- |
| `first_name = 'John' garbage` | ✅ accepted | ❌ error |
| `age > 30 40` | ✅ accepted | ❌ error |
| `first_name = 'John')` | ✅ accepted | ❌ error |
| `1 = 1` | ✅ accepted | ❌ error |

**Migration:** none required for valid queries. If you relied on the lenient
behavior, fix the offending inputs — they were never valid.

### 2. Negated operators now set `IsNot` instead of wrapping in `UnaryOperatorNode` (AST change)

`NOT IN`, `NOT BETWEEN`, `NOT SIMILAR TO`, and `NOT DISTINCT FROM` previously
produced a `UnaryOperatorNode{Operator: NOT}` wrapping the inner node (whose own
`IsNot` was always `false`). They now return the operator node directly with
`IsNot = true`, matching how `RegexMatchNode` already worked. `NOT LIKE` now
produces a `BinaryOperatorNode` with `Operator = TokenOperatorNotLike` instead of
a wrapped `TokenOperatorLike`.

```go
// Before: type-switch had to unwrap the NOT node
if u, ok := node.(*qfv.UnaryOperatorNode); ok && u.Operator == qfv.TokenOperatorNot {
    if in, ok := u.X.(*qfv.InNode); ok { /* negated IN */ }
}

// After: the node carries its own negation
if in, ok := node.(*qfv.InNode); ok && in.IsNot { /* negated IN */ }
```

The `IsNot` fields on `InNode`, `BetweenNode`, `SimilarToNode`, and `DistinctNode`
are now meaningful, and `Type()` returns the corresponding `NodeTypeNotIn`,
`NodeTypeNotBetween`, `NodeTypeNotSimilarTo`, or `NodeTypeNotDistinct`. The
`String()` methods now render the `NOT`/`IS NOT` form correctly.

> Note: a standalone leading `NOT` (e.g. `NOT (age > 30)`) is unchanged — it
> still produces a `UnaryOperatorNode`.

### 3. `DistinctNode` gained a `Value` field (AST change)

`DISTINCT FROM <value>` previously discarded the compared value. `DistinctNode`
now has a `Value Node` field holding it (`nil` when absent).

```go
type DistinctNode struct {
    Field Node
    Value Node // NEW: the value in "field IS [NOT] DISTINCT FROM value"
    IsNot bool
}
```

### 4. `Parse` returns a joined error, not a single formatted string (error shape change)

`FilterParser.Parse` now returns `errors.Join(...)` of `*QFVFilterError` values
rather than one `*QFVFilterError` whose message embedded the others. Use
`errors.As`/`errors.Is` to inspect causes (see [Error Handling](#error-handling)).
Code that only checked `err != nil` is unaffected.

### 5. `FilterParser` is now safe for concurrent use

A single `*FilterParser` (like `*SortParser` and `*FieldsParser`) can now be
created once and shared across goroutines; per-request state is no longer stored
on the parser. No API change — this simply removes a data race.
