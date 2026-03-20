package medparse

import "fmt"

// IndexError is returned when a 1-based index is out of range.
type IndexError struct {
	Type  string // "field", "component", or "sub-component"
	Index int
	Max   int
}

func (e *IndexError) Error() string {
	return fmt.Sprintf("%s index %d out of range (1..%d)", e.Type, e.Index, e.Max)
}

// KeyError is returned when a segment name is not found.
type KeyError struct {
	Name string
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("segment '%s' not found", e.Name)
}

// ParseError is returned when message parsing fails.
type ParseError struct {
	Msg string
}

func (e *ParseError) Error() string {
	return e.Msg
}
