package qfv

import (
	"fmt"
	"strings"
)

type QFVSortError struct {
	Field   string
	Message string
}

func (e *QFVSortError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("error on field '%s': %s", e.Field, e.Message)
	}

	return fmt.Sprintf("error: %s", e.Message)
}

// SortDirection represents the sorting direction in sort expressions
type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

func (sd SortDirection) String() string {
	return string(sd)
}

// SortFieldNode represents a single field in the sort expression
type SortFieldNode struct {
	Field     string
	Direction SortDirection
}

func (n SortFieldNode) Type() NodeType {
	return NodeTypeSortField
}

// SortNode represents the sort part of the query
type SortNode struct {
	Fields []SortFieldNode
}

func (n SortNode) Type() NodeType {
	return NodeTypeSort
}

// SortParser parses the query parameter for sorting
type SortParser struct {
	allowedFields map[string]struct{}
	// allowedDirections is the set of permitted sort directions. A nil map means
	// both ASC and DESC are allowed (the default).
	allowedDirections map[SortDirection]struct{}
}

// SortOption configures a SortParser at construction time.
type SortOption func(*SortParser)

// WithAllowedDirections restricts sorting to the given directions. When the
// option is not used, both ASC and DESC are allowed.
func WithAllowedDirections(dirs ...SortDirection) SortOption {
	return func(p *SortParser) {
		if p.allowedDirections == nil {
			p.allowedDirections = make(map[SortDirection]struct{})
		}
		for _, d := range dirs {
			p.allowedDirections[d] = struct{}{}
		}
	}
}

// NewSortParser creates a new parser with the allowed fields for sorting. By
// default both ASC and DESC are permitted; pass WithAllowedDirections to
// restrict them.
func NewSortParser(allowedFields []string, opts ...SortOption) *SortParser {
	sortFields := make(map[string]struct{}, len(allowedFields))

	for _, f := range allowedFields {
		sortFields[f] = struct{}{}
	}

	p := &SortParser{
		allowedFields: sortFields,
	}
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// directionAllowed reports whether dir is permitted. A nil allow-list permits all.
func (p *SortParser) directionAllowed(dir SortDirection) bool {
	if p.allowedDirections == nil {
		return true
	}
	_, ok := p.allowedDirections[dir]
	return ok
}

// Parse parses the sort parameter
func (p *SortParser) Parse(input string) (SortNode, error) {
	if input == "" {
		return SortNode{}, &QFVSortError{Message: "empty input expression"}
	}

	parts := strings.Split(input, ",")
	fields := make([]SortFieldNode, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return SortNode{}, &QFVSortError{Field: part, Message: "empty sort expression"}
		}

		sortParts := strings.Fields(part)
		if len(sortParts) == 0 {
			return SortNode{}, &QFVSortError{Field: part, Message: "invalid sort expression"}
		}

		if len(sortParts) > 2 {
			return SortNode{}, &QFVSortError{Field: part, Message: "too many sort expressions"}
		}

		fieldName := sortParts[0]
		if _, exists := p.allowedFields[fieldName]; !exists {
			return SortNode{}, &QFVSortError{Field: fieldName, Message: "field not allowed for sorting"}
		}

		direction := SortAsc
		if len(sortParts) == 1 {
			return SortNode{}, &QFVSortError{Field: fieldName, Message: "missing sort direction after field"}
		}

		if len(sortParts) > 1 {
			dirStr := strings.ToUpper(sortParts[1])

			switch dirStr {
			case SortDesc.String():
				direction = SortDesc
			case SortAsc.String():
				direction = SortAsc
			default:
				return SortNode{}, &QFVSortError{Field: fieldName, Message: "invalid sort direction"}
			}
		}

		if !p.directionAllowed(direction) {
			return SortNode{}, &QFVSortError{Field: fieldName, Message: fmt.Sprintf("sort direction %q is not allowed", direction)}
		}

		fields = append(fields, SortFieldNode{
			Field:     fieldName,
			Direction: direction,
		})
	}

	return SortNode{Fields: fields}, nil
}
