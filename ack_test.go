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

func TestBuildACKAndErrorSegment(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ackMsg, err := msg.BuildACK("AE", "Validation error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ackMsg.Segments) != 2 {
		t.Fatalf("expected 2 segments in ACK, got %d", len(ackMsg.Segments))
	}

	msaCode, _ := ackMsg.Get("MSA-1")
	if msaCode != "AE" {
		t.Errorf("expected AE, got %s", msaCode)
	}

	// Add ERR segment to ACK message
	ackMsg.AddErrorSegment("E", "101", "Required field missing", "PID-3-1")
	if len(ackMsg.Segments) != 3 {
		t.Fatalf("expected 3 segments after ERR, got %d", len(ackMsg.Segments))
	}

	errLoc, _ := ackMsg.Get("ERR-1")
	if errLoc != "PID-3-1" {
		t.Errorf("expected PID-3-1, got %s", errLoc)
	}
	errSev, _ := ackMsg.Get("ERR-3")
	if errSev != "E" {
		t.Errorf("expected E, got %s", errSev)
	}
}
