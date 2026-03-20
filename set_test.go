package medparse

import "testing"

func TestSetFieldLevel(t *testing.T) {
	msg, _ := Parse(sampleADT)

	err := msg.Set("PID-8", "F")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-8")
	if val != "F" {
		t.Errorf("expected 'F', got '%s'", val)
	}
}

func TestSetComponentLevel(t *testing.T) {
	msg, _ := Parse(sampleADT)

	err := msg.Set("PID-5-1", "SMITH")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-5-1")
	if val != "SMITH" {
		t.Errorf("expected 'SMITH', got '%s'", val)
	}

	// Other components should be unchanged.
	val, _ = msg.Get("PID-5-2")
	if val != "JOHN" {
		t.Errorf("expected 'JOHN', got '%s'", val)
	}
}

func TestSetSubComponentLevel(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||ID&CHECK^^^AUTH"
	msg, _ := Parse(raw)

	err := msg.Set("PID-3-1-2", "NEWCHECK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-3-1-2")
	if val != "NEWCHECK" {
		t.Errorf("expected 'NEWCHECK', got '%s'", val)
	}
}

func TestSetAutoExtendFields(t *testing.T) {
	msg, _ := Parse(sampleADT)

	// PID only has ~8 fields, set field 20.
	err := msg.Set("PID-20", "EXTENDED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-20")
	if val != "EXTENDED" {
		t.Errorf("expected 'EXTENDED', got '%s'", val)
	}
}

func TestSetAutoExtendComponents(t *testing.T) {
	msg, _ := Parse(sampleADT)

	// PID-5 has 3 components, set component 6.
	err := msg.Set("PID-5-6", "SUFFIX")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-5-6")
	if val != "SUFFIX" {
		t.Errorf("expected 'SUFFIX', got '%s'", val)
	}
}

func TestSetMissingSegment(t *testing.T) {
	msg, _ := Parse(sampleADT)
	err := msg.Set("ZZZ-1", "value")
	if err == nil {
		t.Error("expected error for missing segment")
	}
}

func TestSetInvalidPath(t *testing.T) {
	msg, _ := Parse(sampleADT)
	err := msg.Set("PID", "value")
	if err == nil {
		t.Error("expected error for path without field index")
	}
}

func TestSetRoundtrip(t *testing.T) {
	msg, _ := Parse(sampleADT)

	msg.Set("PID-5-1", "SMITH")
	msg.Set("PID-5-2", "JANE")

	// Re-serialize and re-parse.
	raw := msg.String()
	msg2, err := Parse(raw)
	if err != nil {
		t.Fatalf("roundtrip parse error: %v", err)
	}

	val, _ := msg2.Get("PID-5-1")
	if val != "SMITH" {
		t.Errorf("roundtrip expected 'SMITH', got '%s'", val)
	}
	val, _ = msg2.Get("PID-5-2")
	if val != "JANE" {
		t.Errorf("roundtrip expected 'JANE', got '%s'", val)
	}
}
