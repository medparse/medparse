# medparse 🩺

High-performance HL7v2 message parser and toolkit for Go. Zero dependencies.

## Features

- **Fast** — 3.2μs per message, 10K segments in 11ms with zero-copy fast paths
- **Complete** — Full HL7v2 hierarchy: Message → Segment → Field → Component → Sub-component
- **Read & Write** — Terser-style `Get("PID-5-1")`, `Set("PID-5-1", "SMITH")`, and `GetAll("OBX-5")`
- **Field & Segment Repetitions** — Access repeating fields and segments via `PID-3(1)-1` or dot syntax `PID-5.1`
- **Message Builder** — Create messages from scratch via `NewMessage()`, `AddSegment()`, `InsertSegment()`, and `Clone()`
- **Streaming & Batch** — Parse batches from `io.Reader` using memory-efficient `BatchScanner`
- **Roundtrippable** — Parse → modify → `msg.String()` re-serializes with delimiter escaping integrity
- **Mapping** — Declarative field mapping layer with fallback paths to handle differences between EHRs
- **MLLP-ready** — Automatic framing detection, stripping, `ScanMLLP` (bufio.SplitFunc), and streaming `MLLPReader`/`MLLPWriter`
- **Escape-aware** — Full bidirectional encoding/decoding of HL7 escape sequences (`\F\`, `\S\`, `\T\`, `\R\`, `\E\`, `\.br\`)
- **Validating** — Required-segment validation per message type with custom rule registry (`RegisterRequiredSegments`)
- **Timestamps** — Bidirectional parsing and formatting (`FormatTimestamp`, `FormatTimestampWithTZ`, `FormatDate`)
- **Serializable** — Built-in JSON serialization
- **Zero dependencies** — Go standard library only

## Installation

```bash
go get github.com/medparse/medparse
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/medparse/medparse"
)

func main() {
	msg, err := medparse.Parse(
		"MSH|^~\\&|EPIC|HOSPITAL|RECV|FAC|20260318||ADT^A01^ADT_A01|MSG001|P|2.5.1\r" +
			"PID|1||MRN12345^^^MRN~DEA67890^^^DEA||DOE^JANE^M^^DR||19850315|F\r" +
			"PV1|1|I|4EAST^401^1^^^N||||1234^SMITH^ROBERT^J^^^MD\r" +
			"DG1|1||I10^Essential Hypertension^ICD10||20260318|A",
	)
	if err != nil {
		log.Fatal(err)
	}

	// Terser-style read (supports - and . notation)
	lastName, _ := msg.Get("PID-5-1")   // "DOE"
	firstName, _ := msg.Get("PID-5.2")  // "JANE"
	msgType, _ := msg.Get("MSH-9-1")    // "ADT"
	fmt.Println(lastName, firstName, msgType)

	// Field repetitions
	firstID, _ := msg.Get("PID-3(0)-1")  // "MRN12345"
	secondID, _ := msg.Get("PID-3(1)-1") // "DEA67890"
	fmt.Println("IDs:", firstID, secondID)

	// Terser-style write
	msg.Set("PID-5-1", "SMITH")
	msg.Set("PID-3(1)-1", "NEWDEA")

	// Re-serialize back to HL7
	raw := msg.String()

	// Convenience methods
	et, trig, _ := msg.MessageType()  // "ADT", "A01"
	cid, _ := msg.ControlID()          // "MSG001"

	// Segment iteration
	msg.EachSegment("DG1", func(i int, seg *medparse.Segment) error {
		f, _ := seg.Field(3)
		fmt.Println(i, f.Components[0].Value)
		return nil
	})

	// Validation
	if err := msg.Validate(); err != nil {
		log.Println("invalid:", err)
	}

	// ACK generation (returns string or *Message)
	ack, _ := msg.ACK("AA", "")
	ackMsg, _ := msg.BuildACK("AA", "")

	fmt.Println(et, trig, cid, len(raw), len(ack), ackMsg.Segments[1].Name)
}
```

## API

### Parsing & Streaming

```go
msg, err := medparse.Parse(raw)               // single message string
msg, err := medparse.ParseReader(reader)       // single message from io.Reader
msgs, err := medparse.ParseBatch(raw)         // batch string
msgs, err := medparse.ParseBatchReader(r)     // batch from io.Reader

// Streaming batch scanner for large multi-message files
scanner := medparse.NewBatchScanner(file)
for scanner.Scan() {
    msg := scanner.Message()
    // process msg...
}
if err := scanner.Err(); err != nil {
    log.Fatal(err)
}
```

### Reading & Writing

```go
msg.Get("PID-5-1")                     // terser read
msg.Get("PID-5.1")                     // dot notation
msg.Get("OBX(1)-5")                    // segment repetition (0-based)
msg.Get("PID-3(1)-1")                  // field repetition (0-based)
msg.GetAll("OBX-5")                    // retrieve all values matching path
msg.Set("PID-5-1", "SMITH")            // terser write (auto-extends)
msg.Set("PID-3(1)-1", "DEA123")        // write specific field repetition
```

### Message Construction & Mutation

```go
// Build messages from scratch
msg := medparse.NewMessage("ADT", "A01", "2.5.1")
pid := msg.AddSegment("PID", "1", "", "MRN123^^^MRN", "", "DOE^JOHN")
pv1 := msg.AddSegment("PV1", "1", "I", "4EAST")

// Insert or remove segments
msg.InsertSegment(1, evnSeg)
msg.RemoveSegment(1)
msg.RemoveSegmentsByName("OBX")

// Deep copy
clone := msg.Clone()
```

### Timestamps

```go
ts, err := medparse.ParseTimestamp(raw)                      // HL7 timestamp → time.Time
d, err := medparse.ParseDate(raw)                            // HL7 date → time.Time (date only)
str := medparse.FormatTimestamp(t, medparse.PrecisionSecond) // time.Time → "20230315143022"
tzStr := medparse.FormatTimestampWithTZ(t)                   // time.Time → "20230315143022-0500"
dStr := medparse.FormatDate(t)                               // time.Time → "20230315"
```

### Escaping

```go
escaped := medparse.Escape("Note with | and ^ chars", nil)   // "Note with \F\ and \S\ chars"
unescaped := medparse.Unescape(escaped, nil)                 // "Note with | and ^ chars"
```

### MLLP Streaming (TCP Sockets)

```go
// Wrap / strip
framed := medparse.WrapMLLPString(hl7Msg)
raw := medparse.StripMLLP(framed)

// Read / write over TCP net.Conn
reader := medparse.NewMLLPReader(conn)
writer := medparse.NewMLLPWriter(conn)

msgBytes, err := reader.ReadMessage()
err = writer.WriteString(ackMsg.String())

// Or use bufio.Scanner directly with ScanMLLP
scanner := bufio.NewScanner(conn)
scanner.Split(medparse.ScanMLLP)
for scanner.Scan() {
    msg, _ := medparse.Parse(scanner.Text())
}
```

### Validation

```go
err := msg.Validate()                                  // check required segments for message type
err := msg.ValidateRequiredFields("PID-3-1", "PID-5-1") // validate field presence

// Register custom rules for site-specific or custom message types
medparse.RegisterRequiredSegments("CUSTOM_C01", []string{"MSH", "PID", "Z01"})
```

### Mapping (EHR differences)

```go
import "github.com/medparse/medparse/mapping"

// Fallback path support: try first, fallback to second if missing or empty
epic := mapping.FieldMap{
	"last_name": "PID-5-1",
	"mrn":       "PID-3(0)-1|PID-3-1",
}
val, _ := epic.Get(msg, "last_name")
val := epic.GetOrDefault(msg, "middle_name", "N/A")

// Extractor for complex logic and multi-item extraction
ext := mapping.NewExtractor(epic).
	WithFunc("primary_dx", mapping.SegmentWhere("DG1", 6, "A", 3, 1))

allDx := mapping.SegmentWhereAll("DG1", 6, "A", 3, 1)
dxCodes, _ := allDx(msg)
```

## Benchmarks

Apple M2 (arm64):

```
BenchmarkParse-8               360463       3265 ns/op     5860 B/op    100 allocs/op
BenchmarkParseLarge-8             100   11311098 ns/op 17549896 B/op 240062 allocs/op
BenchmarkGet-8               21028850         59 ns/op       64 B/op      1 allocs/op
BenchmarkSet-8                 459480       2641 ns/op     5012 B/op     89 allocs/op
BenchmarkParseTimestamp-8    10320708        114 ns/op       32 B/op      4 allocs/op
BenchmarkEscape-8             6390405        188 ns/op       80 B/op      1 allocs/op
BenchmarkBuildACK-8            951093       1336 ns/op     3480 B/op     45 allocs/op
BenchmarkBatchScanner-8         13063      89621 ns/op   182059 B/op   3153 allocs/op
```

## License

MIT
