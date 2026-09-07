package medparse

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// TestProperty_EscapeRoundtrip verifies that for any input string,
// Unescape(Escape(s)) reproduces the original string (with normalized newlines).
func TestProperty_EscapeRoundtrip(t *testing.T) {
	enc := DefaultEncodingChars()

	f := func(s string) bool {
		// Normalize newlines since HL7 \.br\ unescapes to \n
		normalized := strings.ReplaceAll(s, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")

		escaped := Escape(normalized, &enc)
		unescaped := Unescape(escaped, &enc)

		return unescaped == normalized
	}

	cfg := &quick.Config{MaxCount: 1000}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Escape roundtrip property failed: %v", err)
	}
}

// TestProperty_TerserGetSet verifies that setting a value at any valid path
// always allows retrieving the exact same value.
func TestProperty_TerserGetSet(t *testing.T) {
	// Restrict values to non-empty printable characters without delimiters
	f := func(fieldNum uint8, compNum uint8, val string) bool {
		fIdx := int(fieldNum%15) + 1
		cIdx := int(compNum%8) + 1
		val = strings.Map(func(r rune) rune {
			if r < 32 || r > 126 || r == '|' || r == '^' || r == '~' || r == '\\' || r == '&' {
				return -1
			}
			return r
		}, val)
		if len(val) == 0 {
			val = "TEST"
		}

		msg := NewMessage("ADT", "A01")
		msg.AddSegment("PID")
		path := fmt.Sprintf("PID-%d-%d", fIdx, cIdx)

		if err := msg.Set(path, val); err != nil {
			return false
		}

		got, err := msg.Get(path)
		return err == nil && got == val
	}

	cfg := &quick.Config{MaxCount: 500}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Terser Get/Set property failed: %v", err)
	}
}

// TestProperty_TimestampRoundtrip verifies that formatting and then parsing
// a timestamp preserves the original time.Time value (to second precision).
func TestProperty_TimestampRoundtrip(t *testing.T) {
	f := func(sec int64) bool {
		// Bound to realistic timestamp range: 1970 to 2038
		if sec < 0 {
			sec = -sec
		}
		sec = sec % 2147483647
		orig := time.Unix(sec, 0).UTC()

		formatted := FormatTimestamp(orig, PrecisionSecond)
		parsed, err := ParseTimestamp(formatted)
		if err != nil {
			return false
		}

		return parsed.Equal(orig)
	}

	cfg := &quick.Config{MaxCount: 1000}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Timestamp roundtrip property failed: %v", err)
	}
}

// TestProperty_MLLPStreamRoundtrip verifies that any sequence of messages
// written to an MLLPWriter can be read back identically by an MLLPReader.
func TestProperty_MLLPStreamRoundtrip(t *testing.T) {
	f := func(messages []string) bool {
		var validMsgs []string
		for _, m := range messages {
			// Strip any MLLP control characters (0x0B, 0x1C) from payload
			cleaned := strings.Map(func(r rune) rune {
				if r == 0x0B || r == 0x1C {
					return -1
				}
				return r
			}, m)
			if len(cleaned) > 0 {
				validMsgs = append(validMsgs, cleaned)
			}
		}

		var buf bytes.Buffer
		writer := NewMLLPWriter(&buf)
		for _, m := range validMsgs {
			if err := writer.WriteString(m); err != nil {
				return false
			}
		}

		reader := NewMLLPReader(&buf)
		for _, expected := range validMsgs {
			got, err := reader.ReadString()
			if err != nil || got != expected {
				return false
			}
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("MLLP stream roundtrip property failed: %v", err)
	}
}

// TestProperty_MessageRoundtrip verifies that building a message, serializing to string,
// and reparsing yields an identical string representation.
func TestProperty_MessageRoundtrip(t *testing.T) {
	f := func(rawPIDName string) bool {
		cleanName := strings.Map(func(r rune) rune {
			if r < 32 || r > 126 || r == '|' || r == '^' || r == '~' || r == '\\' || r == '&' {
				return -1
			}
			return r
		}, rawPIDName)
		if len(cleanName) == 0 {
			cleanName = "DOE"
		}

		msg := NewMessage("ADT", "A01")
		msg.AddSegment("PID", "1", "", "MRN123", "", cleanName)

		s1 := msg.String()
		msg2, err := Parse(s1)
		if err != nil {
			return false
		}
		s2 := msg2.String()

		return s1 == s2
	}

	cfg := &quick.Config{MaxCount: 500}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Message roundtrip property failed: %v", err)
	}
}

// FuzzParse tests that Parse never panics on arbitrary inputs.
func FuzzParse(f *testing.F) {
	f.Add(sampleADT)
	f.Add("MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|1|P|2.5\rPID|1||MRN")
	f.Add("\x0bMSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\x1c\r")
	f.Add("")
	f.Add("garbage")
	f.Add("MSH|")
	f.Add("MSH|^~\\&|")

	f.Fuzz(func(t *testing.T, raw string) {
		// Must not panic
		_, _ = Parse(raw)
	})
}

// FuzzTerser tests that Get and Set never panic on arbitrary paths or values.
func FuzzTerser(f *testing.F) {
	f.Add("PID-5-1", "DOE")
	f.Add("PID-3(1)-1", "VAL")
	f.Add("PID.5.1", "DOE")
	f.Add("OBX(0)-5", "100")
	f.Add("", "")
	f.Add("-1--2", "")
	f.Add("PID(999999999999999999999)-1", "A")

	f.Fuzz(func(t *testing.T, path, val string) {
		msg, err := Parse(sampleADT)
		if err != nil {
			return
		}
		// Must not panic
		_, _ = msg.Get(path)
		_ = msg.Set(path, val)
		_, _ = msg.GetAll(path)
	})
}
