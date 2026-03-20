package medparse

import (
	"strings"
	"testing"
)

func TestACKGeneration(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ack, err := msg.ACK("AA", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(ack, "MSH|^~\\&|") {
		t.Errorf("ACK should start with MSH, got: %s", ack[:20])
	}

	// Verify sender/receiver are swapped.
	if !strings.Contains(ack, "|RECV|FAC|SENDER|FAC|") {
		t.Error("ACK should swap sender and receiver")
	}

	// Verify MSA segment.
	if !strings.Contains(ack, "\rMSA|AA|12345|") {
		t.Errorf("ACK should contain MSA with AA and control ID, got: %s", ack)
	}

	// Verify ACK control ID.
	if !strings.Contains(ack, "ACK12345") {
		t.Error("ACK should contain ACK12345 as control ID")
	}
}

func TestACKWithErrorCode(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ack, err := msg.ACK("AE", "Something went wrong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(ack, "MSA|AE|12345|Something went wrong") {
		t.Errorf("unexpected ACK: %s", ack)
	}
}
