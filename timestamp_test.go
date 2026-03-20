package medparse

import (
	"testing"
	"time"
)

func TestParseTimestampFull(t *testing.T) {
	ts, err := ParseTimestamp("20230315143022")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Year() != 2023 || ts.Month() != time.March || ts.Day() != 15 {
		t.Errorf("unexpected date: %v", ts)
	}
	if ts.Hour() != 14 || ts.Minute() != 30 || ts.Second() != 22 {
		t.Errorf("unexpected time: %v", ts)
	}
}

func TestParseTimestampDateOnly(t *testing.T) {
	ts, err := ParseTimestamp("20230315")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Year() != 2023 || ts.Month() != time.March || ts.Day() != 15 {
		t.Errorf("unexpected date: %v", ts)
	}
	if ts.Hour() != 0 {
		t.Errorf("expected hour 0, got %d", ts.Hour())
	}
}

func TestParseTimestampWithFractional(t *testing.T) {
	ts, err := ParseTimestamp("20230315143022.123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 123 → 123000 microseconds → 123000000 nanoseconds
	expectedNs := 123000 * 1000
	if ts.Nanosecond() != expectedNs {
		t.Errorf("expected %d ns, got %d", expectedNs, ts.Nanosecond())
	}
}

func TestParseTimestampWithFractional4Digits(t *testing.T) {
	ts, err := ParseTimestamp("20230315143022.1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1234 → 123400 microseconds → 123400000 nanoseconds
	expectedNs := 123400 * 1000
	if ts.Nanosecond() != expectedNs {
		t.Errorf("expected %d ns, got %d", expectedNs, ts.Nanosecond())
	}
}

func TestParseTimestampPositiveTZ(t *testing.T) {
	ts, err := ParseTimestamp("20230101120000+0500")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, offset := ts.Zone()
	expectedOffset := 5 * 60 * 60 // +0500 in seconds
	if offset != expectedOffset {
		t.Errorf("expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestParseTimestampNegativeTZ(t *testing.T) {
	ts, err := ParseTimestamp("20230101120000-0700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, offset := ts.Zone()
	expectedOffset := -7 * 60 * 60 // -0700 in seconds
	if offset != expectedOffset {
		t.Errorf("expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestParseTimestampTooShort(t *testing.T) {
	_, err := ParseTimestamp("abc")
	if err == nil {
		t.Error("expected error for short timestamp")
	}
}

func TestParseDateBasic(t *testing.T) {
	d, err := ParseDate("20230315")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Year() != 2023 || d.Month() != time.March || d.Day() != 15 {
		t.Errorf("unexpected date: %v", d)
	}
}

func TestParseDateTooShort(t *testing.T) {
	_, err := ParseDate("2023")
	if err == nil {
		t.Error("expected error for short date")
	}
}

func TestSplitTimezonePositive(t *testing.T) {
	dt, tz := splitTimezone("20230101120000+0500")
	if dt != "20230101120000" {
		t.Errorf("expected '20230101120000', got '%s'", dt)
	}
	if tz == nil || *tz != 300 {
		t.Errorf("expected 300, got %v", tz)
	}
}

func TestSplitTimezoneNegative(t *testing.T) {
	dt, tz := splitTimezone("20230101120000-0700")
	if dt != "20230101120000" {
		t.Errorf("expected '20230101120000', got '%s'", dt)
	}
	if tz == nil || *tz != -420 {
		t.Errorf("expected -420, got %v", tz)
	}
}

func TestSplitTimezoneNone(t *testing.T) {
	dt, tz := splitTimezone("20230101120000")
	if dt != "20230101120000" {
		t.Errorf("expected '20230101120000', got '%s'", dt)
	}
	if tz != nil {
		t.Errorf("expected nil, got %v", tz)
	}
}
