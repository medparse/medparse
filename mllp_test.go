package medparse

import "testing"

func TestDetectMLLP(t *testing.T) {
	framed := "\x0bMSH|...\x1c\r"
	if !IsMLLPFramed([]byte(framed)) {
		t.Error("expected MLLP detection")
	}
}

func TestDetectMLLPNoTrailingCR(t *testing.T) {
	framed := "\x0bMSH|...\x1c"
	if !IsMLLPFramed([]byte(framed)) {
		t.Error("expected MLLP detection")
	}
}

func TestNotMLLP(t *testing.T) {
	if IsMLLPFramed([]byte("MSH|...")) {
		t.Error("should not detect MLLP")
	}
}

func TestStripMLLP(t *testing.T) {
	framed := "\x0bMSH|^~\\&|SENDER\x1c\r"
	if got := StripMLLP(framed); got != "MSH|^~\\&|SENDER" {
		t.Errorf("expected 'MSH|^~\\&|SENDER', got '%s'", got)
	}
}

func TestStripMLLPNoCR(t *testing.T) {
	framed := "\x0bMSH|^~\\&|SENDER\x1c"
	if got := StripMLLP(framed); got != "MSH|^~\\&|SENDER" {
		t.Errorf("expected 'MSH|^~\\&|SENDER', got '%s'", got)
	}
}

func TestStripNoMLLPPassthrough(t *testing.T) {
	raw := "MSH|^~\\&|SENDER"
	if got := StripMLLP(raw); got != raw {
		t.Errorf("expected passthrough, got '%s'", got)
	}
}

func TestShortData(t *testing.T) {
	if IsMLLPFramed([]byte("AB")) {
		t.Error("should not detect MLLP for short data")
	}
	if got := StripMLLP("AB"); got != "AB" {
		t.Errorf("expected 'AB', got '%s'", got)
	}
}

func TestMLLPFramedParse(t *testing.T) {
	raw := "\x0bMSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN\x1c\r"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(msg.Segments))
	}
}
