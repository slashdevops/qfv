package qfv

import (
	"strings"
	"testing"
)

// Each row of this file was run against MySQL 8.4.13 with its default
// sql_mode (ONLY_FULL_GROUP_BY, STRICT_TRANS_TABLES, NO_ZERO_IN_DATE,
// NO_ZERO_DATE, ERROR_FOR_DIVISION_BY_ZERO, NO_ENGINE_SUBSTITUTION: no
// NO_BACKSLASH_ESCAPES), as SELECT count(*) FROM t WHERE (<input>), on a
// table with the columns id INT, name TEXT, price DECIMAL(10,2) and active
// BOOLEAN; what it answered is in the comment. The statement was prepared
// from a hex literal, so that only the server parsed it.

// TestTheMySQLDialectReadsStringsAsMySQLDoes: MySQL reads a backslash in a
// string as an escape, and PostgreSQL's rules read it as a character. The
// filter below is one comparison of name with a string to PostgreSQL's
// reading, which the field allow-list ["name"] lets through; MySQL 8.4,
// with a newline after it as in a SQL template, ran it as a comparison of
// name with a single quote, then id > 0, and the rest of the line a comment:
// every row, on a field the allow-list does not have. In MySQL's dialect it
// is refused.
func TestTheMySQLDialectReadsStringsAsMySQLDoes(t *testing.T) {
	t.Parallel()

	const smuggled = `name = '\'' OR id > 0 #'`

	// PostgreSQL's reading: one comparison, which PostgreSQL 18 runs as one.
	node, err := NewFilterParser([]string{"name"}).Parse(smuggled)
	if err != nil {
		t.Fatalf("PostgreSQL 18's dialect: %v", err)
	}

	if _, ok := node.(*BinaryOperatorNode); !ok {
		t.Fatalf("PostgreSQL 18's dialect read %T, want one comparison", node)
	}

	// MySQL's reading: id is a field of its own, and # is no part of the
	// grammar.
	_, err = NewFilterParser([]string{"name"}, WithDialect(DialectMySQL)).Parse(smuggled)
	if err == nil {
		t.Fatalf("MySQL 8.4's dialect accepted %q, which MySQL runs with a predicate on id", smuggled)
	}

	if !strings.Contains(err.Error(), "error on field 'id': field not allowed") {
		t.Errorf("MySQL 8.4's dialect: %v, want id refused as a field the allow-list does not have", err)
	}

	// The same strings, one token each, as MySQL reads them.
	for input, want := range map[string]string{
		`name = 'a\'b'`:   `'a\'b'`, // one string, a'b: 0 rows (the row holds a\'b)
		`name = 'a\\b'`:   `'a\\b'`, // one string, a\b
		`name = 'a''b'`:   `'a''b'`, // the doubled quote, as in every dialect
		`name LIKE 'x\%'`: `'x\%'`,  // 1 row: \% is a literal %
	} {
		node, err := NewFilterParser([]string{"name"}, WithDialect(DialectMySQL)).Parse(input)
		if err != nil {
			t.Errorf("Parse(%q): %v", input, err)

			continue
		}

		if right := node.(*BinaryOperatorNode).Right.(*LiteralNode).Text; right != want {
			t.Errorf("Parse(%q): the string is %s, want %s", input, right, want)
		}
	}

	// A string that ends on its backslash is not closed: 'a\' is the
	// start of a string to MySQL.
	if _, err := NewFilterParser([]string{"name"}, WithDialect(DialectMySQL)).Parse(`name = 'a\'`); err == nil {
		t.Error(`MySQL 8.4's dialect accepted name = 'a\', an unclosed string to MySQL`)
	}

	// Nor is one whose input ends on the backslash itself.
	if _, err := NewFilterParser([]string{"name"}, WithDialect(DialectMySQL)).Parse(`name = 'a\`); err == nil {
		t.Error(`MySQL 8.4's dialect accepted name = 'a\, an unclosed string`)
	}

	// NewLexer, which callers use directly, reads strings as it always did.
	l := NewLexer(`'a\'`)
	l.Parse()

	if tok := l.Next(); tok.Type != TokenString || tok.Value != `'a\'` {
		t.Errorf("NewLexer read %v, want the PostgreSQL string 'a\\'", tok)
	}
}

func TestTheMySQLDialect(t *testing.T) {
	t.Parallel()

	parser := NewFilterParser([]string{"id", "name", "price", "active"}, WithDialect(DialectMySQL))

	if DialectMySQL.String() != "MySQL 8.4" {
		t.Errorf("String() = %q, want MySQL 8.4", DialectMySQL)
	}

	for _, input := range []string{
		"id = 1", "id <> 1", "id != 1", "id < 2 AND id >= 1", // 1, 2, 2, 1
		"id = 1 OR NOT active IS TRUE",                 // 3
		"id BETWEEN 1 AND 2", "id NOT BETWEEN 1 AND 2", // 2, 1
		"id IN (1, 2)", "id NOT IN (1, 2)", // 2, 1
		"name LIKE 'x%'", "name NOT LIKE 'x%'", // 2, 1
		"name LIKE 'x!%' ESCAPE '!'",       // 1
		"name NOT LIKE 'x!%' ESCAPE '!'",   // 2
		"name LIKE 'x%' ESCAPE ''",         // 2
		`name LIKE 'x\%' ESCAPE '\\'`,      // 1: '\\' is one backslash
		"name IS NULL", "name IS NOT NULL", // 0, 3
		"active IS TRUE", "active IS NOT FALSE", "active IS UNKNOWN", // 1, 2, 1
		"active = TRUE", "price = 1.5", "price = -2", "price = 1e3", // 1, 0, 0, 0
		"name = 'it''s'", // 0
	} {
		t.Run("accepted: "+input, func(t *testing.T) {
			t.Parallel()

			if _, err := parser.Parse(input); err != nil {
				t.Errorf("Parse(%q): %v, want it accepted: MySQL 8.4 runs it", input, err)
			}
		})
	}

	for _, tt := range []struct {
		input string
		says  string
	}{
		// syntax error near 'ILIKE …', '~~ …', 'SIMILAR TO …', '~ …'
		{input: "name ILIKE 'x%'", says: `operator "ILIKE" is not part of the MySQL 8.4 dialect`},
		{input: "name NOT ILIKE 'x%'", says: `operator "ILIKE" is not part of the MySQL 8.4 dialect`},
		{input: "name ~~* 'x%'", says: `operator "ILIKE" is not part of the MySQL 8.4 dialect`},
		{input: "name ~~ 'x%'", says: "~~ is not part of the MySQL 8.4 dialect: write LIKE"},
		{input: "name !~~ 'x%'", says: "!~~ is not part of the MySQL 8.4 dialect: write NOT LIKE"},
		{input: "name SIMILAR TO 'x%'", says: `operator "SIMILAR TO" is not part of the MySQL 8.4 dialect`},
		{input: "name NOT SIMILAR TO 'x%'", says: `operator "SIMILAR TO" is not part of the MySQL 8.4 dialect`},
		{input: "name ~ '^x'", says: `operator "~" is not part of the MySQL 8.4 dialect`},
		{input: "name !~ '^x'", says: `operator "~" is not part of the MySQL 8.4 dialect`},
		// syntax error near 'ISNULL)', 'NOTNULL)'
		{input: "name ISNULL", says: "ISNULL is not part of the MySQL 8.4 dialect: write IS NULL"},
		{input: "name NOTNULL", says: "NOTNULL is not part of the MySQL 8.4 dialect: write IS NOT NULL"},
		// syntax error near 'DISTINCT FROM 1)'
		{input: "id IS DISTINCT FROM 1", says: `operator "IS DISTINCT FROM" is not part of the MySQL 8.4 dialect`},
		{input: "id IS NOT DISTINCT FROM 1", says: `operator "IS DISTINCT FROM" is not part of the MySQL 8.4 dialect`},
		// syntax error near '2 AND 1)'
		{input: "id BETWEEN SYMMETRIC 2 AND 1", says: "BETWEEN SYMMETRIC is not part of the MySQL 8.4 dialect: write BETWEEN, the lower bound first"},
		{input: "id BETWEEN ASYMMETRIC 1 AND 2", says: "BETWEEN ASYMMETRIC is not part of the MySQL 8.4 dialect: write BETWEEN"},
		// Incorrect arguments to ESCAPE (1210)
		{input: "name LIKE 'x%' ESCAPE '!!'", says: "must be empty or one character"},
	} {
		t.Run("refused: "+tt.input, func(t *testing.T) {
			t.Parallel()

			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) accepted, want it refused: MySQL 8.4 does not run it", tt.input)
			}

			if !strings.Contains(err.Error(), tt.says) {
				t.Errorf("Parse(%q) = %q, want it to say %q", tt.input, err.Error(), tt.says)
			}
		})
	}
}

// One quote, written doubled, is one character in every dialect: an ESCAPE
// string of four single quotes is a quote, not an empty string. MySQL 8.4 and
// PostgreSQL 18 each answered 1 row for the filter below (the row 'x%').
func TestADoubledQuoteIsOneEscapeCharacter(t *testing.T) {
	t.Parallel()

	for _, d := range []Dialect{DialectPostgreSQL, DialectSQLStandard, DialectMySQL} {
		if _, err := NewFilterParser([]string{"name"}, WithDialect(d)).Parse("name LIKE 'x''%' ESCAPE ''''"); err != nil {
			t.Errorf("%s: %v, want ESCAPE '''' accepted as one quote", d, err)
		}
	}
}

// TestNoConnectionCharacterSetReadsAStringDifferently: MySQL reads a string
// in the connection's character set, and in gbk, gb18030, sjis, cp932 and
// big5 a backslash (0x5C) can be the last byte of a character that starts
// with any byte from 0x81. With SET NAMES gbk, MySQL 8.4.13 answered 1 row
// for the first filter below: it read the bytes of 丁 and the backslash as
// characters, the backslash escaped nothing, and id = 2 ran. With utf8mb4 it
// answered 0. A backslash right after a character outside ASCII is refused.
func TestNoConnectionCharacterSetReadsAStringDifferently(t *testing.T) {
	t.Parallel()

	parser := NewFilterParser([]string{"name"}, WithDialect(DialectMySQL))

	for _, input := range []string{
		`name = '丁\' OR id = 2 #'`,     // gbk, gb18030, sjis, cp932: 1 row, on id
		"name = '\xbf\\' OR id > 0 #'", // gbk, gb18030: every row
		"name = '\xc3\\' OR id > 0 #'", // gbk, gb18030: every row
		`name = 'é\\'`,                 // the same rule for \\
		`name = '\丁\'x'`,               // after an escaped character too
	} {
		_, err := parser.Parse(input)
		if err == nil {
			t.Errorf("Parse(%q) accepted: a multi-byte connection character set reads the backslash as part of the character before it", input)

			continue
		}

		if !strings.Contains(err.Error(), "a backslash right after a character outside ASCII") {
			t.Errorf("Parse(%q) = %q, want it to say why", input, err.Error())
		}
	}

	// An ASCII character between them reads the same in every character set.
	for _, input := range []string{`name = '丁 \'x'`, `name = '丁'`, `name = 'é'`, `name = 'a\'丁'`} {
		if _, err := parser.Parse(input); err != nil {
			t.Errorf("Parse(%q): %v", input, err)
		}
	}

	// PostgreSQL's strings have no backslash escapes, and nothing to refuse.
	if _, err := NewFilterParser([]string{"name"}).Parse(`name = '丁\'`); err != nil {
		t.Errorf("PostgreSQL 18's dialect: %v", err)
	}
}

// What MySQL 8.4 refuses of an ESCAPE string, the parser refuses: \% and \_
// are two characters outside a pattern ("Incorrect arguments to ESCAPE"; \x
// is one, and ran), and a byte that is not UTF-8 is no character (the same
// error, with a utf8mb4 connection).
func TestTheMySQLEscapeCharacterIsOneCharacterToMySQL(t *testing.T) {
	t.Parallel()

	parser := NewFilterParser([]string{"name"}, WithDialect(DialectMySQL))

	for input, says := range map[string]string{
		`name LIKE 'x%' ESCAPE '\%'`:     "must be empty or one character",
		`name LIKE 'x%' ESCAPE '\_'`:     "must be empty or one character",
		"name LIKE 'x%' ESCAPE '\xc3'":   "not valid UTF-8",
		"name LIKE 'x%' ESCAPE '\\\xff'": "not valid UTF-8",
	} {
		_, err := parser.Parse(input)
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("Parse(%q) = %v, want it to say %q", input, err, says)
		}
	}

	for _, input := range []string{`name LIKE 'x%' ESCAPE '\x'`, "name LIKE 'x%' ESCAPE '�'"} {
		if _, err := parser.Parse(input); err != nil {
			t.Errorf("Parse(%q): %v", input, err)
		}
	}

	// Not UTF-8 is refused in PostgreSQL's dialect too: PostgreSQL refuses
	// the statement's encoding.
	if _, err := NewFilterParser([]string{"name"}).Parse("name LIKE 'x%' ESCAPE '\xc3'"); err == nil {
		t.Error("PostgreSQL 18's dialect accepted an escape that is not UTF-8")
	}
}

// A string's Value is its text with one quote off each end: Trim took every
// quote, and a string that ends on an escaped quote lost it.
func TestAStringsValueKeepsItsLastQuote(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		dialect Dialect
		input   string
		value   string
	}{
		{DialectMySQL, `name = 'a\''`, `a\'`},
		{DialectPostgreSQL, `name = ''''`, `''`},
		{DialectPostgreSQL, `name = 'it''s'`, `it''s`},
	} {
		node, err := NewFilterParser([]string{"name"}, WithDialect(tt.dialect)).Parse(tt.input)
		if err != nil {
			t.Fatalf("%s: Parse(%q): %v", tt.dialect, tt.input, err)
		}

		if got := node.(*BinaryOperatorNode).Right.(*LiteralNode).Value; got != tt.value {
			t.Errorf("%s: Parse(%q): Value = %q, want %q", tt.dialect, tt.input, got, tt.value)
		}
	}
}
