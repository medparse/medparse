package mapping

import (
	"os"
	"path/filepath"
	"testing"

	medparse "github.com/medparse/medparse"
)

const testMsg = "MSH|^~\\&|EPIC|HOSPITAL|RECV|FAC|20230101||ADT^A01|12345|P|2.5\r" +
	"EVN|A01\r" +
	"PID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\r" +
	"PV1|1|I|4EAST^401^1||||1234^SMITH^ROBERT^^^MD\r" +
	"DG1|1||I10^Essential Hypertension^ICD10||20230101|A\r" +
	"DG1|2||M54.5^Low Back Pain^ICD10||20230101|W"

func parseTestMsg(t *testing.T) *medparse.Message {
	t.Helper()
	msg, err := medparse.Parse(testMsg)
	if err != nil {
		t.Fatalf("failed to parse test message: %v", err)
	}
	return msg
}

// ---------------------------------------------------------------------------
// FieldMap tests
// ---------------------------------------------------------------------------

func TestFieldMapGet(t *testing.T) {
	fm := FieldMap{
		"last_name":  "PID-5-1",
		"first_name": "PID-5-2",
		"mrn":        "PID-3-1",
	}

	msg := parseTestMsg(t)

	val, err := fm.Get(msg, "last_name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", val)
	}

	val, _ = fm.Get(msg, "mrn")
	if val != "MRN123" {
		t.Errorf("expected 'MRN123', got '%s'", val)
	}
}

func TestFieldMapGetMissing(t *testing.T) {
	fm := FieldMap{}
	msg := parseTestMsg(t)

	_, err := fm.Get(msg, "nonexistent")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestFieldMapGetAll(t *testing.T) {
	fm := FieldMap{
		"last_name":  "PID-5-1",
		"first_name": "PID-5-2",
		"gender":     "PID-8",
	}

	msg := parseTestMsg(t)
	result := fm.GetAll(msg)

	if result["last_name"] != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", result["last_name"])
	}
	if result["first_name"] != "JOHN" {
		t.Errorf("expected 'JOHN', got '%s'", result["first_name"])
	}
	if result["gender"] != "M" {
		t.Errorf("expected 'M', got '%s'", result["gender"])
	}
}

func TestFieldMapSet(t *testing.T) {
	fm := FieldMap{
		"last_name": "PID-5-1",
	}

	msg := parseTestMsg(t)

	err := fm.Set(msg, "last_name", "SMITH")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := msg.Get("PID-5-1")
	if val != "SMITH" {
		t.Errorf("expected 'SMITH', got '%s'", val)
	}
}

func TestFieldMapMerge(t *testing.T) {
	base := FieldMap{
		"last_name": "PID-5-1",
		"mrn":       "PID-3-1",
	}
	override := FieldMap{
		"mrn":    "PID-18-1", // override
		"gender": "PID-8",    // new
	}

	merged := base.Merge(override)

	if merged["last_name"] != "PID-5-1" {
		t.Error("base key should be preserved")
	}
	if merged["mrn"] != "PID-18-1" {
		t.Error("override should win")
	}
	if merged["gender"] != "PID-8" {
		t.Error("new key should be added")
	}
}

func TestLoadFieldMapJSON(t *testing.T) {
	jsonContent := `{
		"last_name": "PID-5-1",
		"first_name": "PID-5-2",
		"mrn": "PID-3-1"
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "mapping.json")
	os.WriteFile(path, []byte(jsonContent), 0644)

	fm, err := LoadFieldMap(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm["last_name"] != "PID-5-1" {
		t.Errorf("expected 'PID-5-1', got '%s'", fm["last_name"])
	}
	if len(fm) != 3 {
		t.Errorf("expected 3 keys, got %d", len(fm))
	}
}

func TestParseFieldMapJSON(t *testing.T) {
	data := []byte(`{"mrn": "PID-3-1"}`)
	fm, err := ParseFieldMapJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["mrn"] != "PID-3-1" {
		t.Errorf("expected 'PID-3-1', got '%s'", fm["mrn"])
	}
}

// ---------------------------------------------------------------------------
// Extractor tests
// ---------------------------------------------------------------------------

func TestExtractorSimple(t *testing.T) {
	e := NewExtractor(FieldMap{
		"last_name": "PID-5-1",
		"mrn":       "PID-3-1",
	})

	msg := parseTestMsg(t)

	val, err := e.Get(msg, "last_name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", val)
	}
}

func TestExtractorCustomFunc(t *testing.T) {
	e := NewExtractor(FieldMap{
		"last_name": "PID-5-1",
	}).WithFunc("attending_md", FirstSegmentField("PV1", 7, 1))

	msg := parseTestMsg(t)

	val, err := e.Get(msg, "attending_md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "1234" {
		t.Errorf("expected '1234', got '%s'", val)
	}
}

func TestExtractorFuncOverridesFieldMap(t *testing.T) {
	e := NewExtractor(FieldMap{
		"diagnosis": "DG1(0)-3-1", // would get first DG1
	}).WithFunc("diagnosis", func(msg *medparse.Message) (string, error) {
		return "CUSTOM_RESULT", nil
	})

	msg := parseTestMsg(t)

	val, _ := e.Get(msg, "diagnosis")
	if val != "CUSTOM_RESULT" {
		t.Errorf("expected func to override FieldMap, got '%s'", val)
	}
}

func TestSegmentWhere(t *testing.T) {
	e := NewExtractor(FieldMap{}).
		WithFunc("primary_dx", SegmentWhere("DG1", 6, "A", 3, 1))

	msg := parseTestMsg(t)

	val, err := e.Get(msg, "primary_dx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "I10" {
		t.Errorf("expected 'I10' (admitting dx), got '%s'", val)
	}
}

func TestSegmentWhereWorkingDx(t *testing.T) {
	e := NewExtractor(FieldMap{}).
		WithFunc("working_dx", SegmentWhere("DG1", 6, "W", 3, 1))

	msg := parseTestMsg(t)

	val, _ := e.Get(msg, "working_dx")
	if val != "M54.5" {
		t.Errorf("expected 'M54.5' (working dx), got '%s'", val)
	}
}

func TestSegmentWhereNotFound(t *testing.T) {
	e := NewExtractor(FieldMap{}).
		WithFunc("final_dx", SegmentWhere("DG1", 6, "F", 3, 1))

	msg := parseTestMsg(t)

	_, err := e.Get(msg, "final_dx")
	if err == nil {
		t.Error("expected error when no matching segment found")
	}
}

func TestExtractorGetAll(t *testing.T) {
	e := NewExtractor(FieldMap{
		"last_name": "PID-5-1",
	}).WithFunc("primary_dx", SegmentWhere("DG1", 6, "A", 3, 1))

	msg := parseTestMsg(t)
	result := e.GetAll(msg)

	if result["last_name"] != "DOE" {
		t.Errorf("expected 'DOE', got '%s'", result["last_name"])
	}
	if result["primary_dx"] != "I10" {
		t.Errorf("expected 'I10', got '%s'", result["primary_dx"])
	}
}
