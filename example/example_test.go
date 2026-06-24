package example

import (
	"strings"
	"testing"

	qfv "github.com/slashdevops/qfv"
)

func TestParsers(t *testing.T) {
	// Define the common allowed fields for all test cases
	allowedFields := []string{"first_name", "last_name", "email", "created_at", "updated_at"}

	// Test SortParser
	t.Run("SortParser", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			wantErr  bool
			expected []struct {
				field     string
				direction string
			}
		}{
			{
				name:    "valid single sort",
				input:   "first_name ASC",
				wantErr: false,
				expected: []struct {
					field     string
					direction string
				}{
					{field: "first_name", direction: "ASC"},
				},
			},
			{
				name:    "valid multiple sort",
				input:   "first_name ASC,created_at DESC",
				wantErr: false,
				expected: []struct {
					field     string
					direction string
				}{
					{field: "first_name", direction: "ASC"},
					{field: "created_at", direction: "DESC"},
				},
			},
			{
				name:     "invalid field",
				input:    "invalid_field ASC",
				wantErr:  true,
				expected: nil,
			},
			{
				name:     "invalid direction",
				input:    "first_name ASCENDING",
				wantErr:  true,
				expected: nil,
			},
			{
				name:     "empty input",
				input:    "",
				wantErr:  true,
				expected: nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				parser := qfv.NewSortParser(allowedFields)
				result, err := parser.Parse(tc.input)

				// Check if error matches expectation
				if (err != nil) != tc.wantErr {
					t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
					return
				}

				// If no error expected, verify the results
				if !tc.wantErr {
					if len(result.Fields) != len(tc.expected) {
						t.Errorf("Expected %d sort fields, got %d", len(tc.expected), len(result.Fields))
						return
					}

					for i, expectedField := range tc.expected {
						if result.Fields[i].Field != expectedField.field {
							t.Errorf("Expected field %s, got %s", expectedField.field, result.Fields[i].Field)
						}
						if string(result.Fields[i].Direction) != expectedField.direction {
							t.Errorf("Expected direction %s, got %s", expectedField.direction, result.Fields[i].Direction)
						}
					}
				}
			})
		}
	})

	// Test FieldsParser
	t.Run("FieldsParser", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			wantErr  bool
			expected []string
		}{
			{
				name:     "valid single field",
				input:    "first_name",
				wantErr:  false,
				expected: []string{"first_name"},
			},
			{
				name:     "valid multiple fields",
				input:    "first_name, last_name, email",
				wantErr:  false,
				expected: []string{"first_name", "last_name", "email"},
			},
			{
				name:     "invalid field",
				input:    "first_name, invalid_field",
				wantErr:  true,
				expected: nil,
			},
			{
				name:     "empty input",
				input:    "",
				wantErr:  true,
				expected: nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				parser := qfv.NewFieldsParser(allowedFields)
				result, err := parser.Parse(tc.input)

				// Check if error matches expectation
				if (err != nil) != tc.wantErr {
					t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
					return
				}

				// If no error expected, verify the results
				if !tc.wantErr {
					if len(result.Fields) != len(tc.expected) {
						t.Errorf("Expected %d fields, got %d", len(tc.expected), len(result.Fields))
						return
					}

					for i, expected := range tc.expected {
						if result.Fields[i] != expected {
							t.Errorf("Expected field %s, got %s", expected, result.Fields[i])
						}
					}
				}
			})
		}
	})

	// Test FilterParser
	t.Run("FilterParser", func(t *testing.T) {
		testCases := []struct {
			name    string
			input   string
			wantErr bool
		}{
			{
				name:    "simple equality",
				input:   "first_name = 'John'",
				wantErr: false,
			},
			{
				name:    "complex filter with AND and OR",
				input:   "first_name = 'John' AND last_name = 'Doe' OR email = 'example@example.com'",
				wantErr: false,
			},
			{
				name:    "complex filter with parentheses",
				input:   "(first_name = 'John' AND last_name = 'Doe') OR (email = 'example@example.com')",
				wantErr: false,
			},
			{
				name:    "complex filter with date comparison",
				input:   "first_name = 'John' AND (created_at > '2023-01-01' OR updated_at < '2023-12-31')",
				wantErr: false,
			},
			{
				name:    "invalid field",
				input:   "invalid_field = 'value'",
				wantErr: true,
			},
			{
				name:    "syntax error - missing value",
				input:   "first_name =",
				wantErr: true,
			},
			{
				name:    "syntax error - missing operator",
				input:   "first_name 'John'",
				wantErr: true,
			},
			{
				name:    "empty input",
				input:   "",
				wantErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				parser := qfv.NewFilterParser(allowedFields)
				_, err := parser.Parse(tc.input)

				// Check if error matches expectation
				if (err != nil) != tc.wantErr {
					t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
				}
			})
		}
	})
}

func TestMain(t *testing.T) {
	// Integration test simulating the main function
	t.Run("IntegrationTest", func(t *testing.T) {
		// Define the allowed fields for your API (same as in main)
		allowedFields := []string{"first_name", "last_name", "email", "created_at", "updated_at"}

		// Create parsers
		sortParser := qfv.NewSortParser(allowedFields)
		fieldsParser := qfv.NewFieldsParser(allowedFields)
		filterParser := qfv.NewFilterParser(allowedFields)

		// Example inputs (same as in main)
		sortInput := "first_name ASC,created_at DESC"
		fieldsInput := "first_name, last_name, email"
		filterInput := "first_name = 'John' AND last_name = 'Doe' OR email = 'example@example.com' AND (created_at > '2023-01-01' OR updated_at < '2023-12-31')"

		// Test sort parser
		sortNode, err := sortParser.Parse(sortInput)
		if err != nil {
			t.Fatalf("Sort parsing error: %v", err)
		}

		// Verify sort results
		expectedSortFields := []struct {
			field     string
			direction string
		}{
			{field: "first_name", direction: "ASC"},
			{field: "created_at", direction: "DESC"},
		}

		if len(sortNode.Fields) != len(expectedSortFields) {
			t.Errorf("Expected %d sort fields, got %d", len(expectedSortFields), len(sortNode.Fields))
		} else {
			for i, expected := range expectedSortFields {
				if sortNode.Fields[i].Field != expected.field || string(sortNode.Fields[i].Direction) != expected.direction {
					t.Errorf("Sort field %d: expected %s %s, got %s %s",
						i, expected.field, expected.direction,
						sortNode.Fields[i].Field, string(sortNode.Fields[i].Direction))
				}
			}
		}

		// Test fields parser
		fieldsNode, err := fieldsParser.Parse(fieldsInput)
		if err != nil {
			t.Fatalf("Fields parsing error: %v", err)
		}

		// Verify fields results
		expectedFields := []string{"first_name", "last_name", "email"}
		if len(fieldsNode.Fields) != len(expectedFields) {
			t.Errorf("Expected %d fields, got %d", len(expectedFields), len(fieldsNode.Fields))
		} else {
			for i, expected := range expectedFields {
				trimmed := strings.TrimSpace(fieldsNode.Fields[i])
				if trimmed != expected {
					t.Errorf("Field %d: expected '%s', got '%s'", i, expected, trimmed)
				}
			}
		}

		// Test filter parser
		_, err = filterParser.Parse(filterInput)
		if err != nil {
			t.Fatalf("Filter parsing error: %v", err)
		}

		// We've verified the filter parser works without error
		// A more detailed check of the filter structure would require access to internal node details
	})
}
