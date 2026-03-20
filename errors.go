package medparse

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is.
var (
	// ErrParse indicates a message parsing failure.
	ErrParse = errors.New("parse error")

	// ErrNotFound indicates a segment name was not found.
	ErrNotFound = errors.New("not found")

	// ErrIndex indicates a 1-based index was out of range.
	ErrIndex = errors.New("index out of range")
)

// IndexError is returned when a 1-based index is out of range.
type IndexError struct {
	Type  string // "field", "component", or "sub-component"
	Index int
	Max   int
}

func (e *IndexError) Error() string {
	return fmt.Sprintf("%s index %d out of range (1..%d)", e.Type, e.Index, e.Max)
}

func (e *IndexError) Unwrap() error {
	return ErrIndex
}

// KeyError is returned when a segment name is not found.
type KeyError struct {
	Name string
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("segment '%s' not found", e.Name)
}

func (e *KeyError) Unwrap() error {
	return ErrNotFound
}

// ParseError is returned when message parsing fails.
type ParseError struct {
	Msg string
}

func (e *ParseError) Error() string {
	return e.Msg
}

func (e *ParseError) Unwrap() error {
	return ErrParse
}

// ValidationError describes a validation failure for a message.
type ValidationError struct {
	MessageType string
	Missing     []string // segment names that are required but missing
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("message type %s: missing required segments: %v", e.MessageType, e.Missing)
}

func (e *ValidationError) Unwrap() error {
	return ErrParse
}
