# medparse 🩺

High-performance HL7v2 message parser for Go. Zero dependencies.

## Features

- **Fast** — 8μs per message, 10K segments in 30ms
- **Complete** — Full HL7v2 hierarchy: Message → Segment → Field → Component → Sub-component
- **Read & Write** — Terser-style `Get("PID-5-1")` and `Set("PID-5-1", "SMITH")`
- **Roundtrippable** — Parse → modify → `msg.String()` re-serializes back to HL7
- **Mapping** — Declarative field mapping layer to handle differences between EHRs
- **MLLP-aware** — Automatic detection and stripping of MLLP framing
- **Escape-aware** — Decodes HL7 escape sequences (`\F\`, `\S\`, `\T\`, `\R\`, `\E\`)
- **Validating** — Opt-in required-segment validation per message type
- **Serializable** — Built-in JSON serialization
- **Batch-ready** — Parse FHS/BHS-wrapped multi-message files
- **Zero dependencies** — Standard library only

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
			"PID|1||MRN12345^^^MRN||DOE^JANE^M^^DR||19850315|F\r" +
			"PV1|1|I|4EAST^401^1^^^N||||1234^SMITH^ROBERT^J^^^MD\r" +
			"DG1|1||I10^Essential Hypertension^ICD10||20260318|A",
	)
	if err != nil {
		log.Fatal(err)
	}

	// Terser-style read
	lastName, _ := msg.Get("PID-5-1")   // "DOE"
	msgType, _ := msg.Get("MSH-9-1")    // "ADT"
	fmt.Println(lastName, msgType)

	// Terser-style write
	msg.Set("PID-5-1", "SMITH")

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

	// ACK generation
	ack, _ := msg.ACK("AA", "")

	fmt.Println(et, trig, cid, len(raw), len(ack))
}
```

## API

### Parsing

```go
msg, err := medparse.Parse(raw)             // single message
msgs, err := medparse.ParseBatch(raw)       // batch of messages
ts, err := medparse.ParseTimestamp(raw)      // HL7 timestamp → time.Time
d, err := medparse.ParseDate(raw)           // HL7 timestamp → date only
```

### Reading & Writing

```go
msg.Get("PID-5-1")                   // terser read
msg.Get("OBX(1)-5")                  // segment repetition (0-based)
msg.Set("PID-5-1", "SMITH")          // terser write (auto-extends)
```

### Message Access

```go
msg.Segment("PID")                   // first matching segment
msg.SegmentsByName("OBX")            // all matching segments
msg.EachSegment("OBX", func(i int, seg *Segment) error { ... })
msg.MessageType()                    // ("ADT", "A01")
msg.ControlID()                      // "MSG001"
msg.Version()                        // "2.5.1"
```

### Serialization & Output

```go
msg.String()                         // re-serialize to HL7 pipe format
msg.ToJSON()                         // JSON serialization
msg.ACK("AA", "")                    // generate ACK response
```

### Mapping (EHR differences)

```go
import "github.com/medparse/medparse/mapping"

epic := mapping.FieldMap{
	"last_name": "PID-5-1",
	"mrn":       "PID-3-1",
}
val, _ := epic.Get(msg, "last_name")

// Extractor for complex logic
ext := mapping.NewExtractor(epic).
	WithFunc("primary_dx", mapping.SegmentWhere("DG1", 6, "A", 3, 1))
```

### Validation

```go
err := msg.Validate()                // check required segments for message type
```

### MLLP

```go
medparse.IsMLLPFramed(data)          // detect MLLP framing
medparse.StripMLLP(raw)              // strip framing (Parse does this automatically)
```

### Error Handling

```go
// Sentinel errors for errors.Is
errors.Is(err, medparse.ErrParse)     // parse failures
errors.Is(err, medparse.ErrNotFound)  // missing segment
errors.Is(err, medparse.ErrIndex)     // out of range

// Typed errors for errors.As
var ke *medparse.KeyError
errors.As(err, &ke)                   // ke.Name == "ZZZ"
```

## Benchmarks

```
BenchmarkParse-8               133441     8315 ns/op     7440 B/op    199 allocs/op
BenchmarkParseLarge-8              40  29559748 ns/op 23678806 B/op 570119 allocs/op
BenchmarkGet-8               12444850      105 ns/op       56 B/op      2 allocs/op
BenchmarkSet-8                 183168     6372 ns/op     6200 B/op    168 allocs/op
BenchmarkParseTimestamp-8     6494740      221 ns/op       32 B/op      4 allocs/op
BenchmarkString-8             2960448      399 ns/op      408 B/op     11 allocs/op
```

## License

MIT
