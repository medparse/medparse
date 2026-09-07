package medparse

import (
	"bytes"
	"io"
	"testing"
)

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

func TestWrapMLLP(t *testing.T) {
	msg := "MSH|^~\\&|TEST"
	wrapped := WrapMLLPString(msg)
	expected := "\x0b" + msg + "\x1c\r"
	if wrapped != expected {
		t.Errorf("WrapMLLPString = %q, expected %q", wrapped, expected)
	}
	if !IsMLLPFramed([]byte(wrapped)) {
		t.Error("wrapped string should be identified as MLLP framed")
	}
	stripped := StripMLLP(wrapped)
	if stripped != msg {
		t.Errorf("StripMLLP = %q, expected %q", stripped, msg)
	}
}

func TestMLLPStreamingReaderAndWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewMLLPWriter(&buf)

	msg1 := "MSH|^~\\&|APP1||||20230101||ADT^A01|1|P|2.5"
	msg2 := "MSH|^~\\&|APP2||||20230101||ADT^A02|2|P|2.5"

	err := writer.WriteString(msg1)
	if err != nil {
		t.Fatalf("writer error: %v", err)
	}
	err = writer.WriteString(msg2)
	if err != nil {
		t.Fatalf("writer error: %v", err)
	}

	reader := NewMLLPReader(&buf)

	got1, err := reader.ReadString()
	if err != nil {
		t.Fatalf("reading msg1: %v", err)
	}
	if got1 != msg1 {
		t.Errorf("expected msg1 %q, got %q", msg1, got1)
	}

	got2, err := reader.ReadString()
	if err != nil {
		t.Fatalf("reading msg2: %v", err)
	}
	if got2 != msg2 {
		t.Errorf("expected msg2 %q, got %q", msg2, got2)
	}

	// Next read should be EOF
	_, err = reader.ReadString()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}
