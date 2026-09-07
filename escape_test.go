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

func TestEscapeBasic(t *testing.T) {
	enc := DefaultEncodingChars()

	cases := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"a|b", `a\F\b`},
		{"a^b", `a\S\b`},
		{"a&b", `a\T\b`},
		{"a~b", `a\R\b`},
		{`a\b`, `a\E\b`},
		{"line1\nline2", `line1\.br\line2`},
		{"line1\r\nline2", `line1\.br\line2`},
		{"a|b^c&d~e\\f\ng", `a\F\b\S\c\T\d\R\e\E\f\.br\g`},
	}

	for _, tc := range cases {
		got := Escape(tc.input, &enc)
		if got != tc.expected {
			t.Errorf("Escape(%q) = %q; expected %q", tc.input, got, tc.expected)
		}
		// Test roundtrip
		roundtrip := Unescape(got, &enc)
		// For \r\n, Unescape decodes \.br\ to \n
		expectedRoundtrip := tc.input
		if tc.input == "line1\r\nline2" {
			expectedRoundtrip = "line1\nline2"
		}
		if roundtrip != expectedRoundtrip {
			t.Errorf("Unescape(Escape(%q)) = %q; expected %q", tc.input, roundtrip, expectedRoundtrip)
		}
	}
}

func TestEscapeNilEnc(t *testing.T) {
	if got := Escape("a|b", nil); got != `a\F\b` {
		t.Errorf("expected 'a\\F\\b', got %q", got)
	}
	if got := Unescape(`a\F\b`, nil); got != "a|b" {
		t.Errorf("expected 'a|b', got %q", got)
	}
}
