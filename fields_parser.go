package qfv

import (
	"fmt"
	"strings"
)

type QFVFieldsError struct {
	Field   string
	Message string
}

func (e *QFVFieldsError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("error on field '%s': %s", e.Field, e.Message)
	}

	return fmt.Sprintf("error: %s", e.Message)
}

// FieldsNode represents the fields part of the query
type FieldsNode struct {
	Fields []string
}

func (n FieldsNode) Type() NodeType {
	return NodeTypeFieldList
}

// FieldsParser parses the query parameter for fields
type FieldsParser struct {
	allowedFieldsFields map[string]struct{}
}

// NewFieldsParser creates a new parser with the allowed fields
func NewFieldsParser(allowedFields []string) *FieldsParser {
	fieldsFields := make(map[string]struct{}, len(allowedFields))

	for _, f := range allowedFields {
		fieldsFields[f] = struct{}{}
	}

	return &FieldsParser{
		allowedFieldsFields: fieldsFields,
	}
}

// Parse parses the fields parameter
func (p *FieldsParser) Parse(input string) (FieldsNode, error) {
	if input == "" {
		return FieldsNode{}, &QFVFieldsError{Message: "empty input expression"}
	}

	parts := strings.Split(input, ",")
	fields := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return FieldsNode{}, &QFVFieldsError{Field: part, Message: "empty field expression"}
		}

		if _, exists := p.allowedFieldsFields[part]; !exists {
			return FieldsNode{}, &QFVFieldsError{Field: part, Message: "unknown field"}
		}

		fields = append(fields, part)
	}

	return FieldsNode{Fields: fields}, nil
}
