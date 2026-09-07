package medparse

import (
	"strings"
	"testing"
)

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

func TestBatchScanner(t *testing.T) {
	raw := "FHS|^~\\&|BATCH\rBHS|^~\\&|BATCH\r" +
		"MSH|^~\\&|S1|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN1\r" +
		"MSH|^~\\&|S2|F|R|F|20230102||ADT^A02|2|P|2.5\rPID|1||MRN2\r" +
		"BTS|2\rFTS|2"

	scanner := NewBatchScanner(strings.NewReader(raw))
	var scanned []*Message
	for scanner.Scan() {
		scanned = append(scanned, scanner.Message())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	if len(scanned) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(scanned))
	}

	app1, _ := scanned[0].SendingApplication()
	if app1 != "S1" {
		t.Errorf("expected S1, got %s", app1)
	}

	app2, _ := scanned[1].SendingApplication()
	if app2 != "S2" {
		t.Errorf("expected S2, got %s", app2)
	}
}
