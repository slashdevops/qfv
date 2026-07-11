package qfv

import (
	"errors"
	"testing"
)

func TestQFVFilterError_Error(t *testing.T) {
	withField := &QFVFilterError{Field: "name", Message: "bad"}
	if got, want := withField.Error(), "error on field 'name': bad"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	noField := &QFVFilterError{Message: "bad"}
	if got, want := noField.Error(), "error: bad"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestParse_JoinedErrorsAreInspectable verifies the aggregated error can be
// unwrapped into *QFVFilterError values via errors.As.
func TestParse_JoinedErrorsAreInspectable(t *testing.T) {
	p := NewFilterParser([]string{"name"})
	_, err := p.Parse("unknown = 'x'")
	if err == nil {
		t.Fatal("expected error for disallowed field")
	}
	var fe *QFVFilterError
	if !errors.As(err, &fe) {
		t.Fatalf("errors.As failed to extract *QFVFilterError from %T", err)
	}
	if fe.Field != "unknown" {
		t.Errorf("Field = %q, want unknown", fe.Field)
	}
}

func TestNode_TypeMethods(t *testing.T) {
	if got := (FieldsNode{}).Type(); got != NodeTypeFieldList {
		t.Errorf("FieldsNode.Type() = %s", got)
	}
	if got := (SortNode{}).Type(); got != NodeTypeSort {
		t.Errorf("SortNode.Type() = %s", got)
	}
	if got := (SortFieldNode{}).Type(); got != NodeTypeSortField {
		t.Errorf("SortFieldNode.Type() = %s", got)
	}
}

// TestFilterParser_ErrorBranches drives the IN/BETWEEN error paths.
func TestFilterParser_ErrorBranches(t *testing.T) {
	p := NewFilterParser([]string{"name", "age"})
	cases := []string{
		"name IN ()",        // empty list
		"name IN ('x',)",    // trailing comma
		"name IN 'x'",       // missing opening paren
		"name IN ('x'",      // missing closing paren
		"age BETWEEN 10",    // missing AND
		"age BETWEEN 10 20", // missing AND keyword
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := p.Parse(in); err == nil {
				t.Errorf("Parse(%q) = nil error, want error", in)
			}
		})
	}
}
