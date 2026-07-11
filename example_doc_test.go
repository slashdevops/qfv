package qfv_test

import (
	"errors"
	"fmt"

	qfv "github.com/slashdevops/qfv"
)

// allowed is the set of fields an API is willing to expose to clients.
var allowed = []string{"first_name", "last_name", "email", "age", "active", "created_at"}

func ExampleFieldsParser() {
	parser := qfv.NewFieldsParser(allowed)

	node, err := parser.Parse("first_name, email")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, f := range node.Fields {
		fmt.Println(f)
	}
	// Output:
	// first_name
	// email
}

func ExampleSortParser() {
	parser := qfv.NewSortParser(allowed)

	node, err := parser.Parse("first_name ASC, created_at DESC")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, f := range node.Fields {
		fmt.Printf("%s %s\n", f.Field, f.Direction)
	}
	// Output:
	// first_name ASC
	// created_at DESC
}

func ExampleFilterParser() {
	parser := qfv.NewFilterParser(allowed)

	node, err := parser.Parse("first_name = 'John' AND age >= 18")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(node.String())
	// Output:
	// ((first_name = 'John') AND (age >= 18))
}

// ExampleFilterParser_postgres shows the PostgreSQL predicates: case-insensitive
// ILIKE, IS [NOT] DISTINCT FROM (null-safe comparison), and IS boolean tests.
func ExampleFilterParser_postgres() {
	parser := qfv.NewFilterParser(allowed)

	node, err := parser.Parse("email ILIKE '%@EXAMPLE.com' AND age IS NOT DISTINCT FROM 30 OR active IS TRUE")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(node.String())
	// Output:
	// (((email ILIKE '%@EXAMPLE.com') AND age IS NOT DISTINCT FROM 30) OR active IS TRUE)
}

// ExampleFilterParser_errorInspection shows how to inspect validation failures.
// Parse aggregates every problem with errors.Join, so errors.As extracts the
// first *QFVFilterError cause.
func ExampleFilterParser_errorInspection() {
	parser := qfv.NewFilterParser(allowed)

	_, err := parser.Parse("secret = 'x'")

	var fe *qfv.QFVFilterError
	if errors.As(err, &fe) {
		fmt.Printf("field %q rejected: %s\n", fe.Field, fe.Message)
	}
	// Output:
	// field "secret" rejected: field not allowed
}
