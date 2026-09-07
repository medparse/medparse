package medparse

import (
	"strings"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage("ADT", "A01")
	if len(msg.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(msg.Segments))
	}
	if msg.Segments[0].Name != "MSH" {
		t.Errorf("expected MSH segment, got %s", msg.Segments[0].Name)
	}

	et, trig, err := msg.MessageType()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != "ADT" || trig != "A01" {
		t.Errorf("expected ADT/A01, got %s/%s", et, trig)
	}

	ver, err := msg.Version()
	if err != nil || ver != "2.5.1" {
		t.Errorf("expected 2.5.1, got %s", ver)
	}

	cid, err := msg.ControlID()
	if err != nil || !strings.HasPrefix(cid, "MSG") {
		t.Errorf("expected MSG prefix in control ID, got %s", cid)
	}
}

func TestNewMessageCustomVersion(t *testing.T) {
	msg := NewMessage("ORU", "R01", "2.3")
	ver, err := msg.Version()
	if err != nil || ver != "2.3" {
		t.Errorf("expected 2.3, got %s", ver)
	}
}

func TestMessageMutation(t *testing.T) {
	msg := NewMessage("ADT", "A01")

	// AddSegment
	pid := msg.AddSegment("PID", "1", "", "MRN123^^^MRN", "", "DOE^JOHN")
	if len(msg.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(msg.Segments))
	}
	if pid.Name != "PID" {
		t.Errorf("expected PID, got %s", pid.Name)
	}

	val, err := msg.Get("PID-5-1")
	if err != nil || val != "DOE" {
		t.Errorf("expected DOE, got %q (err: %v)", val, err)
	}

	// Add another segment
	pv1 := msg.AddSegment("PV1", "1", "I", "4EAST")
	if len(msg.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(msg.Segments))
	}

	// InsertSegment
	evn := Segment{Name: "EVN", Fields: []Field{parseField("A01", &msg.Enc)}}
	err = msg.InsertSegment(1, evn)
	if err != nil {
		t.Fatalf("unexpected insert error: %v", err)
	}
	if msg.Segments[1].Name != "EVN" {
		t.Errorf("expected EVN at index 1, got %s", msg.Segments[1].Name)
	}

	// SegmentsByNamePtr
	ptrs := msg.SegmentsByNamePtr("PID")
	if len(ptrs) != 1 {
		t.Fatalf("expected 1 PID pointer, got %d", len(ptrs))
	}
	ptrs[0].SetField(5, "SMITH^JANE")
	val, err = msg.Get("PID-5-1")
	if err != nil || val != "SMITH" {
		t.Errorf("expected SMITH after mutating via pointer, got %q", val)
	}

	// RemoveSegment
	err = msg.RemoveSegment(1) // remove EVN
	if err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	if msg.Segments[1].Name != "PID" {
		t.Errorf("expected PID back at index 1, got %s", msg.Segments[1].Name)
	}

	// Add multiple OBX and RemoveSegmentsByName
	msg.AddSegment("OBX", "1", "NM", "TEST1")
	msg.AddSegment("OBX", "2", "NM", "TEST2")
	removed := msg.RemoveSegmentsByName("OBX")
	if removed != 2 {
		t.Errorf("expected 2 OBX segments removed, got %d", removed)
	}
	if len(msg.SegmentsByName("OBX")) != 0 {
		t.Error("expected 0 OBX segments remaining")
	}

	// Check serialization of built message
	serialized := msg.String()
	reparsed, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse built message: %v", err)
	}
	val, err = reparsed.Get("PID-5-2")
	if err != nil || val != "JANE" {
		t.Errorf("expected JANE, got %q", val)
	}
	_ = pv1
}

func TestMessageClone(t *testing.T) {
	msg, err := Parse(sampleADT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clone := msg.Clone()
	if len(clone.Segments) != len(msg.Segments) {
		t.Fatalf("expected same segment count")
	}

	// Modify clone
	clone.Set("PID-5-1", "CLONED")
	origVal, _ := msg.Get("PID-5-1")
	cloneVal, _ := clone.Get("PID-5-1")

	if origVal != "DOE" {
		t.Errorf("original message should not be modified, got %s", origVal)
	}
	if cloneVal != "CLONED" {
		t.Errorf("clone should have modified value, got %s", cloneVal)
	}
}

func TestSegmentHelpers(t *testing.T) {
	seg := Segment{Name: "PID"}
	seg.AddField("1")
	seg.AddField("")
	seg.AddField("MRN123")

	if len(seg.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(seg.Fields))
	}

	err := seg.SetField(5, "DOE^JOHN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seg.Fields) != 5 {
		t.Fatalf("expected 5 fields after auto-extend, got %d", len(seg.Fields))
	}

	compVal, err := seg.Component(5, 1)
	if err != nil || compVal != "DOE" {
		t.Errorf("expected DOE, got %q", compVal)
	}

	err = seg.ClearField(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := seg.Field(5)
	if f.Value != "" {
		t.Errorf("expected empty field, got %q", f.Value)
	}
}
