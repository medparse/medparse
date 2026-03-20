package medparse

import "testing"

func TestBatchSimple(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN1\rMSH|^~\\&|S|F|R|F|20230102||ADT^A01|2|P|2.5\rPID|1||MRN2"
	msgs, err := ParseBatch(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestBatchWithFHSWrapper(t *testing.T) {
	raw := "FHS|^~\\&|BATCH\rBHS|^~\\&|BATCH\rMSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN1\rBTS|1\rFTS|1"
	msgs, err := ParseBatch(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestBatchEmpty(t *testing.T) {
	msgs, err := ParseBatch("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestBatchSingle(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN"
	msgs, err := ParseBatch(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}
