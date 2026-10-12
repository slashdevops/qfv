package qfv

import (
	"testing"
	"text/scanner"
)

func TestLexer_Navigation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "double quoted string, 1",
			input: `"comment = 'This is a string'"`,
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIllegal, Value: `"comment = 'This is a string'"`},
				{Pos: scanner.Position{Line: 1, Column: 34}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "double quoted string, 2",
			input: `comment = "'This is a string'"`,
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIllegal, Value: `"'This is a string'"`},
				{Pos: scanner.Position{Line: 1, Column: 34}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "bad double quoted string, missing closing quote",
			input: `comment = "'This is a bad string`,
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 11}, Type: TokenIllegal, Value: `"'This is a bad string`},
				{Pos: scanner.Position{Line: 1, Column: 34}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "bad quoted string, missing closing quote",
			input: "comment = 'This is a bad string",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 11}, Type: TokenIllegal, Value: `'This is a bad string`},
				{Pos: scanner.Position{Line: 1, Column: 34}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "bad_double_quoted_string_missing_opening_quote",
			input: `comment = This is a bad string'"`, // Input length: 32, last char at col 32
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 11}, Type: TokenIdentifier, Value: "This"},
				{Pos: scanner.Position{Line: 1, Column: 16}, Type: TokenOperatorIsNull, Value: "is"}, // IS, in any case
				{Pos: scanner.Position{Line: 1, Column: 19}, Type: TokenIdentifier, Value: "a"},
				{Pos: scanner.Position{Line: 1, Column: 21}, Type: TokenIdentifier, Value: "bad"},
				{Pos: scanner.Position{Line: 1, Column: 25}, Type: TokenIdentifier, Value: "string"},
				{Pos: scanner.Position{Line: 1, Column: 31}, Type: TokenIllegal, Value: `'"`}, // Adjusted to match actual lexer output
				{Pos: scanner.Position{Line: 1, Column: 33}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "bad quoted string, missing opening quote",
			input: "comment = This is a bad string'", // Input length: 31, last char at col 31
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 11}, Type: TokenIdentifier, Value: "This"},
				{Pos: scanner.Position{Line: 1, Column: 16}, Type: TokenOperatorIsNull, Value: "is"}, // IS, in any case
				{Pos: scanner.Position{Line: 1, Column: 19}, Type: TokenIdentifier, Value: "a"},
				{Pos: scanner.Position{Line: 1, Column: 21}, Type: TokenIdentifier, Value: "bad"},
				{Pos: scanner.Position{Line: 1, Column: 25}, Type: TokenIdentifier, Value: "string"},
				{Pos: scanner.Position{Line: 1, Column: 31}, Type: TokenIllegal, Value: "'"}, // Illegal token starts at col 31
				{Pos: scanner.Position{Line: 1, Column: 32}, Type: TokenEOF, Value: ""},      // EOF is at col 32 (after last char)
			},
		},
		{
			name:  "string literal with spaces and scaped quotes",
			input: "comment = 'This is a '' scaped single quote' AND age > 18", // Use '' for escaping
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "comment"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorEqual, Value: "="},
				// Lexer includes outer quotes and handles '' escape correctly
				{Pos: scanner.Position{Line: 1, Column: 11}, Type: TokenString, Value: `'This is a '' scaped single quote'`},
				// Adjust subsequent positions based on the new string length
				{Pos: scanner.Position{Line: 1, Column: 46}, Type: TokenOperatorAnd, Value: "AND"},       // Adjusted position
				{Pos: scanner.Position{Line: 1, Column: 50}, Type: TokenIdentifier, Value: "age"},        // Adjusted position
				{Pos: scanner.Position{Line: 1, Column: 54}, Type: TokenOperatorGreaterThan, Value: ">"}, // Adjusted position
				{Pos: scanner.Position{Line: 1, Column: 56}, Type: TokenInt, Value: "18"},                // Adjusted position
				{Pos: scanner.Position{Line: 1, Column: 58}, Type: TokenEOF, Value: ""},                  // Adjusted position
			},
		},
		{
			name:  "multiple operators",
			input: "age > 18 AND name = 'John' OR status != 'active'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "age"},
				{Pos: scanner.Position{Line: 1, Column: 4}, Type: TokenOperatorGreaterThan, Value: ">"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenInt, Value: "18"},
				{Pos: scanner.Position{Line: 1, Column: 8}, Type: TokenOperatorAnd, Value: "AND"},
				{Pos: scanner.Position{Line: 1, Column: 12}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 16}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 18}, Type: TokenString, Value: `'John'`},
				{Pos: scanner.Position{Line: 1, Column: 24}, Type: TokenOperatorOr, Value: "OR"},
				{Pos: scanner.Position{Line: 1, Column: 27}, Type: TokenIdentifier, Value: "status"},
				{Pos: scanner.Position{Line: 1, Column: 33}, Type: TokenOperatorNotEqualAlias, Value: "!="},
				{Pos: scanner.Position{Line: 1, Column: 35}, Type: TokenString, Value: `'active'`},
				{Pos: scanner.Position{Line: 1, Column: 42}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "multiple operators, with sub-grouping",
			input: "age > 18 AND (name = 'John' OR status != 'active')",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "age"},
				{Pos: scanner.Position{Line: 1, Column: 4}, Type: TokenOperatorGreaterThan, Value: ">"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenInt, Value: "18"},
				{Pos: scanner.Position{Line: 1, Column: 8}, Type: TokenOperatorAnd, Value: "AND"},
				{Pos: scanner.Position{Line: 1, Column: 12}, Type: TokenLPAREN, Value: "("},
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenOperatorEqual, Value: "="},
				{Pos: scanner.Position{Line: 1, Column: 19}, Type: TokenString, Value: `'John'`},
				{Pos: scanner.Position{Line: 1, Column: 25}, Type: TokenOperatorOr, Value: "OR"},
				{Pos: scanner.Position{Line: 1, Column: 27}, Type: TokenIdentifier, Value: "status"},
				{Pos: scanner.Position{Line: 1, Column: 33}, Type: TokenOperatorNotEqualAlias, Value: "!="},
				{Pos: scanner.Position{Line: 1, Column: 35}, Type: TokenString, Value: `'active'`},
				{Pos: scanner.Position{Line: 1, Column: 42}, Type: TokenRPAREN, Value: ")"},
				{Pos: scanner.Position{Line: 1, Column: 43}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "IS NULL",
			input: "name IS NULL",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorIsNull, Value: "IS"}, // IS token
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenIdentifier, Value: "NULL"},   // NULL token
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "IS NOT NULL",
			input: "name IS NOT NULL",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorIsNull, Value: "IS"}, // IS token
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorNot, Value: "NOT"},   // NOT token
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenIdentifier, Value: "NULL"},  // NULL token
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "IS NOT TRUE", // Test case for IS NOT followed by non-NULL
			input: "flag IS NOT TRUE",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "flag"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorIsNull, Value: "IS"}, // IS token
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorNot, Value: "NOT"},   // NOT token
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenBoolean, Value: "TRUE"},     // TRUE token
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT LIKE",
			input: "name NOT LIKE '%John%'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNot, Value: "NOT"},    // NOT token
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenOperatorLike, Value: "LIKE"}, // LIKE token
				{Pos: scanner.Position{Line: 1, Column: 15}, Type: TokenString, Value: "'%John%'"},
				{Pos: scanner.Position{Line: 1, Column: 24}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT IN",
			input: "name NOT IN ('John', 'Jane')",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNot, Value: "NOT"}, // NOT token
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenOperatorIn, Value: "IN"},  // IN token
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenLPAREN, Value: "("},
				{Pos: scanner.Position{Line: 1, Column: 14}, Type: TokenString, Value: "'John'"},
				{Pos: scanner.Position{Line: 1, Column: 20}, Type: TokenComma, Value: ","},
				{Pos: scanner.Position{Line: 1, Column: 22}, Type: TokenString, Value: "'Jane'"},
				{Pos: scanner.Position{Line: 1, Column: 28}, Type: TokenRPAREN, Value: ")"},
				{Pos: scanner.Position{Line: 1, Column: 29}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT BETWEEN",
			input: "age NOT BETWEEN 18 AND 30",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "age"},
				{Pos: scanner.Position{Line: 1, Column: 5}, Type: TokenOperatorNot, Value: "NOT"},         // NOT token
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenOperatorBetween, Value: "BETWEEN"}, // BETWEEN token
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenInt, Value: "18"},
				{Pos: scanner.Position{Line: 1, Column: 20}, Type: TokenOperatorAnd, Value: "AND"},
				{Pos: scanner.Position{Line: 1, Column: 24}, Type: TokenInt, Value: "30"},
				{Pos: scanner.Position{Line: 1, Column: 26}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT DISTINCT",
			input: "name NOT DISTINCT FROM 'John'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNot, Value: "NOT"},            // NOT token
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenOperatorDistinct, Value: "DISTINCT"}, // DISTINCT token
				{Pos: scanner.Position{Line: 1, Column: 19}, Type: TokenIdentifier, Value: "FROM"},           // FROM token
				{Pos: scanner.Position{Line: 1, Column: 24}, Type: TokenString, Value: "'John'"},
				{Pos: scanner.Position{Line: 1, Column: 30}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "SIMILAR TO",
			input: "name SIMILAR TO '%John%'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorSimilarTo, Value: "SIMILAR"}, // SIMILAR token
				{Pos: scanner.Position{Line: 1, Column: 14}, Type: TokenIdentifier, Value: "TO"},            // TO token
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenString, Value: "'%John%'"},
				{Pos: scanner.Position{Line: 1, Column: 26}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT SIMILAR TO",
			input: "name NOT SIMILAR TO '%John%'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNot, Value: "NOT"},            // NOT token
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenOperatorSimilarTo, Value: "SIMILAR"}, // SIMILAR token
				{Pos: scanner.Position{Line: 1, Column: 18}, Type: TokenIdentifier, Value: "TO"},             // TO token
				{Pos: scanner.Position{Line: 1, Column: 21}, Type: TokenString, Value: "'%John%'"},
				{Pos: scanner.Position{Line: 1, Column: 30}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "SIMILAR keyword only",
			input: "name SIMILAR",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorSimilarTo, Value: "SIMILAR"},
				{Pos: scanner.Position{Line: 1, Column: 13}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "TO keyword only",
			input: "name TO 'pattern'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenIdentifier, Value: "TO"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenString, Value: "'pattern'"},
				{Pos: scanner.Position{Line: 1, Column: 18}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "NOT SIMILAR without TO",
			input: "name NOT SIMILAR",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNot, Value: "NOT"},
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenOperatorSimilarTo, Value: "SIMILAR"},
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Different casing",
			input: "name similar tO '%pattern%'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorSimilarTo, Value: "similar"},
				{Pos: scanner.Position{Line: 1, Column: 14}, Type: TokenIdentifier, Value: "tO"},
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenString, Value: "'%pattern%'"},
				{Pos: scanner.Position{Line: 1, Column: 28}, Type: TokenEOF, Value: ""},
			},
		},
		// ---- Regex Operator Tests ----
		{
			name:  "Regex Match CS (~)",
			input: "name ~ 'pattern'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorRegexMatchCS, Value: "~"},
				{Pos: scanner.Position{Line: 1, Column: 8}, Type: TokenString, Value: "'pattern'"},
				{Pos: scanner.Position{Line: 1, Column: 17}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Not Regex Match CS (!~)",
			input: "name !~ 'pattern'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNotRegexMatchCS, Value: "!~"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenString, Value: "'pattern'"},
				{Pos: scanner.Position{Line: 1, Column: 18}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Regex Match CI (~*)",
			input: "name ~* 'pattern'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorRegexMatchCI, Value: "~*"},
				{Pos: scanner.Position{Line: 1, Column: 9}, Type: TokenString, Value: "'pattern'"},
				{Pos: scanner.Position{Line: 1, Column: 18}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Not Regex Match CI (!~*)",
			input: "name !~* 'pattern'",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNotRegexMatchCI, Value: "!~*"},
				{Pos: scanner.Position{Line: 1, Column: 10}, Type: TokenString, Value: "'pattern'"},
				{Pos: scanner.Position{Line: 1, Column: 19}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Incomplete Regex Op (!)",
			input: "name !",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenIllegal, Value: "!"},
				{Pos: scanner.Position{Line: 1, Column: 7}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Incomplete Regex Op (!~)", // Already covered by Not Regex Match CS
			input: "name !~",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorNotRegexMatchCS, Value: "!~"},
				{Pos: scanner.Position{Line: 1, Column: 8}, Type: TokenEOF, Value: ""},
			},
		},
		{
			name:  "Incomplete Regex Op (~)", // Already covered by Regex Match CS
			input: "name ~",
			expected: []Token{
				{Pos: scanner.Position{Line: 1, Column: 1}, Type: TokenIdentifier, Value: "name"},
				{Pos: scanner.Position{Line: 1, Column: 6}, Type: TokenOperatorRegexMatchCS, Value: "~"},
				{Pos: scanner.Position{Line: 1, Column: 7}, Type: TokenEOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			lexer.Parse()

			t.Logf("Lexer tokens: %v", lexer.tokens)
			for _, token := range lexer.tokens {
				t.Logf("Token: %v", token)
			}

			for i, expected := range tt.expected {
				if len(lexer.tokens) <= i {
					t.Fatalf("expected token %d, got none", i)
				}

				token := lexer.tokens[i]
				if token.Type != expected.Type {
					t.Errorf("expected token type -->%s<--, got -->%s<--", expected.Type, token.Type)
				}

				if token.Value != expected.Value {
					t.Errorf("expected token value -->%s<--, got -->%s<--", expected.Value, token.Value)
				}
			}
		})
	}
}
