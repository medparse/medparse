package medparse

import "testing"

func TestTerserBasicField(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("PID-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "DOE^JOHN^M" {
		t.Errorf("expected 'DOE^JOHN^M', got '%s'", val)
	}
}

func TestTerserComponent(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("PID-5-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", val)
	}

	val, err = msg.Get("PID-5-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "JOHN" {
		t.Errorf("expected 'JOHN', got '%s'", val)
	}
}

func TestTerserMSHMessageType(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("MSH-9-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "ADT" {
		t.Errorf("expected 'ADT', got '%s'", val)
	}
}

func TestTerserSegmentOnly(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("PID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "PID" {
		t.Errorf("expected 'PID', got '%s'", val)
	}
}

func TestTerserSegmentRepetition(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rOBX|1|NM|CODE1||100\rOBX|2|NM|CODE2||200"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("OBX(0)-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "100" {
		t.Errorf("expected '100', got '%s'", val)
	}

	val, err = msg.Get("OBX(1)-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "200" {
		t.Errorf("expected '200', got '%s'", val)
	}
}

func TestTerserMissingSegment(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = msg.Get("ZZZ-1")
	if err == nil {
		t.Error("expected error for missing segment")
	}
}

func TestTerserInvalidFieldIndex(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = msg.Get("PID-abc")
	if err == nil {
		t.Error("expected error for invalid field index")
	}
}

func TestTerserSubComponent(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||ID&CHECK^^^AUTH"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("PID-3-1-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "ID" {
		t.Errorf("expected 'ID', got '%s'", val)
	}

	val, err = msg.Get("PID-3-1-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "CHECK" {
		t.Errorf("expected 'CHECK', got '%s'", val)
	}
}
