package medparse

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const sampleADT = "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1"

func TestParseBasicMessage(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(msg.Segments))
	}
	if msg.Segments[0].Name != "MSH" {
		t.Errorf("expected MSH, got %s", msg.Segments[0].Name)
	}
	if msg.Segments[1].Name != "PID" {
		t.Errorf("expected PID, got %s", msg.Segments[1].Name)
	}
	if msg.Segments[2].Name != "PV1" {
		t.Errorf("expected PV1, got %s", msg.Segments[2].Name)
	}
}

func TestMSHFields(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msh := &msg.Segments[0]

	// MSH-1 = field separator
	if msh.Fields[0].Value != "|" {
		t.Errorf("MSH-1 expected '|', got '%s'", msh.Fields[0].Value)
	}
	// MSH-2 = encoding characters
	if msh.Fields[1].Value != "^~\\&" {
		t.Errorf("MSH-2 expected '^~\\&', got '%s'", msh.Fields[1].Value)
	}
	// MSH-3 = sending application
	if msh.Fields[2].Value != "SENDER" {
		t.Errorf("MSH-3 expected 'SENDER', got '%s'", msh.Fields[2].Value)
	}
	// MSH-9 = message type
	if msh.Fields[8].Components[0].Value != "ADT" {
		t.Errorf("MSH-9.1 expected 'ADT', got '%s'", msh.Fields[8].Components[0].Value)
	}
	if msh.Fields[8].Components[1].Value != "A01" {
		t.Errorf("MSH-9.2 expected 'A01', got '%s'", msh.Fields[8].Components[1].Value)
	}
	// MSH-10 = control ID
	if msh.Fields[9].Value != "12345" {
		t.Errorf("MSH-10 expected '12345', got '%s'", msh.Fields[9].Value)
	}
	// MSH-12 = version
	if msh.Fields[11].Value != "2.5" {
		t.Errorf("MSH-12 expected '2.5', got '%s'", msh.Fields[11].Value)
	}
}

func TestPIDPatientName(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid := &msg.Segments[1]

	nameField := &pid.Fields[4] // PID-5
	if nameField.Components[0].Value != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", nameField.Components[0].Value)
	}
	if nameField.Components[1].Value != "JOHN" {
		t.Errorf("expected 'JOHN', got '%s'", nameField.Components[1].Value)
	}
	if nameField.Components[2].Value != "M" {
		t.Errorf("expected 'M', got '%s'", nameField.Components[2].Value)
	}
}

func TestNewlineDelimiter(t *testing.T) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|123|P|2.5\nPID|1||MRN|||DOE^JOHN"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(msg.Segments))
	}
}

func TestCRLFDelimiter(t *testing.T) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|123|P|2.5\r\nPID|1||MRN|||DOE^JOHN"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(msg.Segments))
	}
}

func TestEmptyFields(t *testing.T) {
	raw := "MSH|^~\\&|||||20230101||ADT^A01|123|P|2.5\rPID|1||MRN|||||||"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid := &msg.Segments[1]
	if len(pid.Fields) < 9 {
		t.Errorf("expected >= 9 fields, got %d", len(pid.Fields))
	}
}

func TestRepetition(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN1^^^MRN~DEA1^^^DEA"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid := &msg.Segments[1]
	idField := &pid.Fields[2] // PID-3
	if len(idField.Repetitions) != 2 {
		t.Fatalf("expected 2 repetitions, got %d", len(idField.Repetitions))
	}
	if idField.Repetitions[0].Components[0].Value != "MRN1" {
		t.Errorf("expected 'MRN1', got '%s'", idField.Repetitions[0].Components[0].Value)
	}
	if idField.Repetitions[1].Components[0].Value != "DEA1" {
		t.Errorf("expected 'DEA1', got '%s'", idField.Repetitions[1].Components[0].Value)
	}
}

func TestSubcomponents(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||ID&CHECK^^^AUTH"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid := &msg.Segments[1]
	idField := &pid.Fields[2] // PID-3
	firstComp := &idField.Components[0]
	if len(firstComp.SubComponents) != 2 {
		t.Fatalf("expected 2 sub-components, got %d", len(firstComp.SubComponents))
	}
	if firstComp.SubComponents[0] != "ID" {
		t.Errorf("expected 'ID', got '%s'", firstComp.SubComponents[0])
	}
	if firstComp.SubComponents[1] != "CHECK" {
		t.Errorf("expected 'CHECK', got '%s'", firstComp.SubComponents[1])
	}
}

func TestErrorNoMSH(t *testing.T) {
	_, err := Parse("PID|1||MRN")
	if err == nil {
		t.Error("expected error for non-MSH message")
	}
}

func TestErrorEmpty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestEscapeSequencesInFields(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rOBX|1|ST|CODE||value\\F\\with\\S\\special"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obx := &msg.Segments[1]
	valueField := &obx.Fields[4] // OBX-5
	if valueField.Value != "value|with^special" {
		t.Errorf("expected 'value|with^special', got '%s'", valueField.Value)
	}
}

func TestMessageConvenienceMethods(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	et, trig, err := msg.MessageType()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != "ADT" || trig != "A01" {
		t.Errorf("expected (ADT, A01), got (%s, %s)", et, trig)
	}

	cid, err := msg.ControlID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cid != "12345" {
		t.Errorf("expected '12345', got '%s'", cid)
	}

	ver, err := msg.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "2.5" {
		t.Errorf("expected '2.5', got '%s'", ver)
	}

	app, err := msg.SendingApplication()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app != "SENDER" {
		t.Errorf("expected 'SENDER', got '%s'", app)
	}

	fac, err := msg.SendingFacility()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fac != "FAC" {
		t.Errorf("expected 'FAC', got '%s'", fac)
	}
}

func TestSegmentByName(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seg, err := msg.Segment("PID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seg.Name != "PID" {
		t.Errorf("expected 'PID', got '%s'", seg.Name)
	}

	_, err = msg.Segment("ZZZ")
	if err == nil {
		t.Error("expected error for missing segment")
	}
}

func TestSegmentsByName(t *testing.T) {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rOBX|1|NM|CODE1||100\rOBX|2|NM|CODE2||200"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obxs := msg.SegmentsByName("OBX")
	if len(obxs) != 2 {
		t.Errorf("expected 2 OBX segments, got %d", len(obxs))
	}
}

func TestToJSON(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	j, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(j, "MSH") {
		t.Error("JSON should contain MSH")
	}
	if !strings.Contains(j, "DOE") {
		t.Error("JSON should contain DOE")
	}
}

func TestPerformanceLargeMessage(t *testing.T) {
	// Build a message with 10k segments.
	var b strings.Builder
	b.WriteString("MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5")
	for i := 0; i < 10000; i++ {
		b.WriteString(fmt.Sprintf("\rOBX|%d|NM|CODE-%d||%d|unit|0-100||||F", i, i, i*7))
	}

	raw := b.String()
	start := time.Now()
	msg, err := Parse(raw)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 10001 {
		t.Errorf("expected 10001 segments, got %d", len(msg.Segments))
	}
	if elapsed.Milliseconds() >= 500 {
		t.Errorf("parsing took %dms, expected < 500ms", elapsed.Milliseconds())
	}
}

func TestCustomEncodingChars(t *testing.T) {
	raw := "MSH#^~\\&#S#F#R#F#20230101##ADT^A01#1#P#2.5\rPID#1##MRN"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(msg.Segments))
	}
}

func TestFieldAccess1Based(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid, _ := msg.Segment("PID")
	f, err := pid.Field(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Components[0].Value != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", f.Components[0].Value)
	}

	// Out of range
	_, err = pid.Field(0)
	if err == nil {
		t.Error("expected error for field index 0")
	}
	_, err = pid.Field(999)
	if err == nil {
		t.Error("expected error for field index 999")
	}
}

func TestComponentAccess1Based(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pid, _ := msg.Segment("PID")
	f, _ := pid.Field(5)

	comp, err := f.Component(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Value != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", comp.Value)
	}

	_, err = f.Component(0)
	if err == nil {
		t.Error("expected error for component index 0")
	}
}
