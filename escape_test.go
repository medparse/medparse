package medparse

import "testing"

func TestNoEscapes(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("hello world", &enc); got != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", got)
	}
}

func TestFieldSepEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("before\\F\\after", &enc); got != "before|after" {
		t.Errorf("expected 'before|after', got '%s'", got)
	}
}

func TestComponentSepEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("a\\S\\b", &enc); got != "a^b" {
		t.Errorf("expected 'a^b', got '%s'", got)
	}
}

func TestSubcomponentSepEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("a\\T\\b", &enc); got != "a&b" {
		t.Errorf("expected 'a&b', got '%s'", got)
	}
}

func TestRepetitionSepEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("a\\R\\b", &enc); got != "a~b" {
		t.Errorf("expected 'a~b', got '%s'", got)
	}
}

func TestEscapeCharEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("a\\E\\b", &enc); got != "a\\b" {
		t.Errorf("expected 'a\\b', got '%s'", got)
	}
}

func TestLineBreakEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("line1\\.br\\line2", &enc); got != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got '%s'", got)
	}
}

func TestMultipleEscapes(t *testing.T) {
	enc := DefaultEncodingChars()
	if got := decodeEscapes("a\\F\\b\\S\\c", &enc); got != "a|b^c" {
		t.Errorf("expected 'a|b^c', got '%s'", got)
	}
}

func TestHexEscape(t *testing.T) {
	enc := DefaultEncodingChars()
	// \X0D\ = carriage return
	if got := decodeEscapes("a\\X0D\\b", &enc); got != "a\rb" {
		t.Errorf("expected 'a\\rb', got '%s'", got)
	}
}
