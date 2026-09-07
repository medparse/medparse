package medparse

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkParse(b *testing.B) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1\rDG1|1||I10^Essential Hypertension^ICD10||20230101|A"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLarge(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5")
	for i := 0; i < 10000; i++ {
		buf.WriteString(fmt.Sprintf("\rOBX|%d|NM|CODE-%d||%d|unit|0-100||||F", i, i, i*7))
	}
	raw := buf.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1"
	msg, _ := Parse(raw)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := msg.Get("PID-5-1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSet(b *testing.B) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg, _ := Parse(raw)
		msg.Set("PID-5-1", "SMITH")
	}
}

func BenchmarkParseTimestamp(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseTimestamp("20230315143022.123-0500")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkString(b *testing.B) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1"
	msg, _ := Parse(raw)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.String()
	}
}

func BenchmarkEscape(b *testing.B) {
	enc := DefaultEncodingChars()
	input := "Doctor & Nurse notes with | and ^ and ~ characters."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Escape(input, &enc)
	}
}

func BenchmarkBuildACK(b *testing.B) {
	raw := "MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101120000||ADT^A01|12345|P|2.5\rPID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M\rPV1|1|I|4EAST^401^1"
	msg, _ := Parse(raw)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.BuildACK("AA", "")
	}
}

func BenchmarkBatchScanner(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("FHS|^~\\&|BATCH\rBHS|^~\\&|BATCH")
	for i := 0; i < 50; i++ {
		buf.WriteString(fmt.Sprintf("\rMSH|^~\\&|S%d|F|R|F|20230101||ADT^A01|%d|P|2.5\rPID|1||MRN%d||DOE^JOHN", i, i, i))
	}
	buf.WriteString("\rBTS|50\rFTS|50")
	raw := buf.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewBatchScanner(strings.NewReader(raw))
		count := 0
		for scanner.Scan() {
			count++
		}
	}
}
