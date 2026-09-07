package medparse

import (
	"errors"
	"testing"
)

func TestValidateADTA01Valid(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rEVN|A01\rPID|1||MRN\rPV1|1|I|4EAST"
	msg, _ := Parse(raw)

	err := msg.Validate()
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateADTA01MissingEVN(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN\rPV1|1|I|4EAST"
	msg, _ := Parse(raw)

	err := msg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(ve.Missing) != 1 || ve.Missing[0] != "EVN" {
		t.Errorf("expected missing [EVN], got %v", ve.Missing)
	}
}

func TestValidateORUR01(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ORU^R01|1|P|2.5\rPID|1||MRN\rOBR|1\rOBX|1|NM|CODE||100"
	msg, _ := Parse(raw)

	err := msg.Validate()
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateUnknownMessageType(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ZZZ^Z99|1|P|2.5\rPID|1||MRN"
	msg, _ := Parse(raw)

	// Unknown message type — no rules, should pass.
	err := msg.Validate()
	if err != nil {
		t.Errorf("expected nil for unknown type, got: %v", err)
	}
}

func TestErrorWrapping(t *testing.T) {
	// ParseError wraps ErrParse.
	_, err := Parse("")
	if !errors.Is(err, ErrParse) {
		t.Errorf("expected errors.Is(err, ErrParse), got %v", err)
	}

	// KeyError wraps ErrNotFound.
	msg, _ := Parse(sampleADT)
	_, err = msg.Segment("ZZZ")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}

	// IndexError wraps ErrIndex.
	pid, _ := msg.Segment("PID")
	_, err = pid.Field(999)
	if !errors.Is(err, ErrIndex) {
		t.Errorf("expected errors.Is(err, ErrIndex), got %v", err)
	}

	// errors.As works too.
	var ke *KeyError
	_, err = msg.Segment("ZZZ")
	if !errors.As(err, &ke) {
		t.Errorf("expected errors.As to work for KeyError")
	}
	if ke.Name != "ZZZ" {
		t.Errorf("expected 'ZZZ', got '%s'", ke.Name)
	}
}

func TestRegisterRequiredSegments(t *testing.T) {
	RegisterRequiredSegments("CUSTOM_C01", []string{"MSH", "PID", "Z01"})

	raw := "MSH|^~\\&|S|F|R|F|20230101||CUSTOM^C01|1|P|2.5\rPID|1||MRN"
	msg, _ := Parse(raw)

	err := msg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing Z01 segment")
	}

	// Add Z01 segment and revalidate
	msg.AddSegment("Z01", "CUSTOM_DATA")
	err = msg.Validate()
	if err != nil {
		t.Errorf("expected valid after adding Z01, got: %v", err)
	}
}

func TestValidateRequiredFields(t *testing.T) {
	msg, _ := Parse(sampleADT)

	// Valid fields present
	err := msg.ValidateRequiredFields("MSH-9-1", "PID-5-1")
	if err != nil {
		t.Errorf("expected valid fields, got: %v", err)
	}

	// Missing / empty field
	err = msg.ValidateRequiredFields("PID-5-1", "PID-19")
	if err == nil {
		t.Error("expected error for empty field PID-19")
	}
}
