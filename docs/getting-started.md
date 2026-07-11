# Getting Started

## Install

```bash
go get github.com/slashdevops/qfv@latest
```

## Update

```bash
go get -u github.com/slashdevops/qfv@latest
go mod tidy
```

To pin or move to a specific version:

```bash
go get github.com/slashdevops/qfv@v0.1.0
```

## Overview

QFV (Query Filters Validator) is an Abstract Syntax Tree (AST) parser that
validates and parses three common kinds of query expression:

1. **Fields selection** — which fields to include in the response
2. **Filtering** — conditions used to filter results
3. **Sorting** — the order of returned results

Every parser is built from an **allow-list of fields**, so only fields you
explicitly permit can appear in a query. This is the primary defense against
invalid or malicious input reaching your database.

All three parsers are **safe for concurrent use** — build them once and share
them across goroutines.

## Complete example

```go
package main

import (
    "fmt"
    "log"

    qfv "github.com/slashdevops/qfv"
)

func main() {
    // Fields your API is willing to expose.
    allowedFields := []string{"first_name", "last_name", "email", "created_at", "updated_at"}

    // Build the parsers once; they are safe to reuse concurrently.
    sortParser := qfv.NewSortParser(allowedFields)
    fieldsParser := qfv.NewFieldsParser(allowedFields)
    filterParser := qfv.NewFilterParser(allowedFields)

    // Example inputs (typically query-string parameters).
    sortInput := "first_name ASC, created_at DESC"
    fieldsInput := "first_name, last_name, email"
    filterInput := "first_name = 'John' AND last_name = 'Doe'"

    sortNode, err := sortParser.Parse(sortInput)
    if err != nil {
        log.Fatalf("sort validation error: %v", err)
    }

    fieldsNode, err := fieldsParser.Parse(fieldsInput)
    if err != nil {
        log.Fatalf("fields validation error: %v", err)
    }

    filterNode, err := filterParser.Parse(filterInput)
    if err != nil {
        log.Fatalf("filter validation error: %v", err)
    }

    fmt.Println("Sort fields:")
    for _, f := range sortNode.Fields {
        fmt.Printf("  %s %s\n", f.Field, f.Direction)
    }

    fmt.Println("Requested fields:")
    for _, f := range fieldsNode.Fields {
        fmt.Printf("  %s\n", f)
    }

    // The filter parser returns an AST you can walk or render.
    fmt.Println("Filter AST:", filterNode.String())
}
```

## Next steps

- [Filtering](filtering.md) — the full set of operators and predicates
- [Configuration](configuration.md) — restrict operators and sort directions
- [Error Handling](error-handling.md) — inspecting validation failures
