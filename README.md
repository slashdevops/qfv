# Query Filters Validator (QFV)

[![Go Reference](https://pkg.go.dev/badge/github.com/slashdevops/qfv.svg)](https://pkg.go.dev/github.com/slashdevops/qfv)

A Go library for parsing and validating the query expressions commonly used in
REST APIs and database queries — **fields selection**, **filtering**, and
**sorting**. Each parser is driven by an allow-list of fields, so only fields you
permit can appear in a query, helping secure your endpoints against invalid or
malicious input.

## Install

```bash
go get github.com/slashdevops/qfv@latest
```

Update to the latest release with `go get -u github.com/slashdevops/qfv@latest`.

## Quick start

```go
package main

import (
    "fmt"
    "log"

    qfv "github.com/slashdevops/qfv"
)

func main() {
    allowedFields := []string{"first_name", "last_name", "email", "created_at"}

    // Parsers are safe to build once and reuse across goroutines.
    filterParser := qfv.NewFilterParser(allowedFields)
    sortParser := qfv.NewSortParser(allowedFields)
    fieldsParser := qfv.NewFieldsParser(allowedFields)

    node, err := filterParser.Parse("first_name = 'John' AND created_at > '2023-01-01'")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(node.String()) // ((first_name = 'John') AND (created_at > '2023-01-01'))

    if _, err := sortParser.Parse("first_name ASC, created_at DESC"); err != nil {
        log.Fatal(err)
    }
    if _, err := fieldsParser.Parse("first_name, last_name, email"); err != nil {
        log.Fatal(err)
    }
}
```

## Features

- **Three validators** — fields selection, filtering, and sorting, each gated by
  an allow-list of fields (nested dot-notation like `user.profile.age` supported).
- **Rich, PostgreSQL-flavored filter grammar** — comparison, logical, `IN`,
  `BETWEEN [SYMMETRIC]`, `LIKE`/`ILIKE`, `SIMILAR TO`, POSIX regex, `IS NULL`,
  `IS [NOT] TRUE/FALSE/UNKNOWN`, and `IS [NOT] DISTINCT FROM`.
- **Configurable** — restrict which operators or sort directions are allowed.
- **Returns an AST** you can walk or render, with precise, aggregated errors.
- **Concurrency-safe** parsers.

## Documentation

Full documentation lives in [`docs/`](docs/README.md):

- [Getting Started](docs/getting-started.md) — install, update, complete example
- [Filtering](docs/filtering.md) — the full operator/predicate reference
- [Configuration](docs/configuration.md) — restrict operators and sort directions
- [Error Handling](docs/error-handling.md) — inspecting validation failures
- [Migration Guide](docs/migration.md) — upgrading between versions

API reference and runnable examples are on
[pkg.go.dev](https://pkg.go.dev/github.com/slashdevops/qfv).

## License

See the repository for license details.
