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

func TestTerserDotNotation(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := msg.Get("PID-5.1")
	if err != nil || val != "DOE" {
		t.Errorf("expected DOE, got %q (err: %v)", val, err)
	}

	val, err = msg.Get("PID.5.2")
	if err != nil || val != "JOHN" {
		t.Errorf("expected JOHN, got %q (err: %v)", val, err)
	}
}

func TestTerserFieldRepetition(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN123^^^MRN~DEA456^^^DEA"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First repetition (0-based)
	val, err := msg.Get("PID-3(0)-1")
	if err != nil || val != "MRN123" {
		t.Errorf("expected MRN123, got %q (err: %v)", val, err)
	}

	// Second repetition
	val, err = msg.Get("PID-3(1)-1")
	if err != nil || val != "DEA456" {
		t.Errorf("expected DEA456, got %q (err: %v)", val, err)
	}

	val, err = msg.Get("PID-3(1)")
	if err != nil || val != "DEA456^^^DEA" {
		t.Errorf("expected DEA456^^^DEA, got %q (err: %v)", val, err)
	}

	// Out of range field repetition
	_, err = msg.Get("PID-3(2)-1")
	if err == nil {
		t.Error("expected error for out-of-range field repetition")
	}
}

func TestTerserGetAll(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\r" +
		"PID|1||MRN123^^^MRN~DEA456^^^DEA\r" +
		"OBX|1|NM|GLU||100|mg/dL\r" +
		"OBX|2|NM|WBC||6.5|10*3/uL"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GetAll across segments
	vals, err := msg.GetAll("OBX-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 2 || vals[0] != "100" || vals[1] != "6.5" {
		t.Errorf("expected [100, 6.5], got %v", vals)
	}

	// GetAll across field repetitions
	ids, err := msg.GetAll("PID-3-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "MRN123" || ids[1] != "DEA456" {
		t.Errorf("expected [MRN123, DEA456], got %v", ids)
	}
}
