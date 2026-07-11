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

// ExampleWithAllowedOperatorGroups restricts the grammar to a subset of
// operators. Anything outside the allow-list is rejected at parse time.
func ExampleWithAllowedOperatorGroups() {
	parser := qfv.NewFilterParser(allowed,
		qfv.WithAllowedOperatorGroups(qfv.GroupComparison, qfv.GroupLogical),
		qfv.WithAllowedOperators(qfv.OpIn),
	)

	if _, err := parser.Parse("age >= 18 AND email IN ('a@x.com', 'b@x.com')"); err == nil {
		fmt.Println("comparison + IN: allowed")
	}
	if _, err := parser.Parse("email LIKE '%@x.com'"); err != nil {
		fmt.Println("LIKE:", err)
	}
	// Output:
	// comparison + IN: allowed
	// LIKE: error: operator "LIKE" is not allowed
}

// ExampleWithAllowedDirections forbids DESC sorting.
func ExampleWithAllowedDirections() {
	parser := qfv.NewSortParser(allowed, qfv.WithAllowedDirections(qfv.SortAsc))

	if _, err := parser.Parse("first_name ASC"); err == nil {
		fmt.Println("ASC: allowed")
	}
	if _, err := parser.Parse("first_name DESC"); err != nil {
		fmt.Println("DESC:", err)
	}
	// Output:
	// ASC: allowed
	// DESC: error on field 'first_name': sort direction "DESC" is not allowed
}

// ExampleFilterParser_dottedFields shows nested dot-notation field names.
func ExampleFilterParser_dottedFields() {
	parser := qfv.NewFilterParser([]string{"user.profile.age", "user.name"})

	node, err := parser.Parse("user.profile.age >= 18 AND user.name = 'John'")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(node.String())
	// Output:
	// ((user.profile.age >= 18) AND (user.name = 'John'))
}
