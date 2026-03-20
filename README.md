# medparse 🩺

High-performance HL7v2 message parser for Go. Zero dependencies.

## Features

- **Fast** — Parses 10K segments in ~30ms
- **Complete** — Full HL7v2 hierarchy: Message → Segment → Field → Component → Sub-component
- **Ergonomic** — Terser-style path access (`msg.Get("PID-5-1")`)
- **MLLP-aware** — Automatic detection and stripping of MLLP framing
- **Escape-aware** — Decodes HL7 escape sequences (`\F\`, `\S\`, `\T\`, `\R\`, `\E\`)
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

	// Segment access
	pid, _ := msg.Segment("PID")
	dg1s := msg.SegmentsByName("DG1")

	// Field access (1-indexed, per HL7 spec)
	nameField, _ := pid.Field(5)
	lastName, _ := nameField.Component(1)   // "DOE"
	firstName, _ := nameField.Component(2)  // "JANE"

	// Terser-style shorthand
	val, _ := msg.Get("PID-5-1")   // "DOE"
	mtype, _ := msg.Get("MSH-9-1") // "ADT"

	// MSH convenience methods
	et, trig, _ := msg.MessageType() // "ADT", "A01"
	cid, _ := msg.ControlID()        // "MSG001"
	ver, _ := msg.Version()           // "2.5.1"

	// Serialization
	jsonStr, _ := msg.ToJSON()

	// ACK generation
	ack, _ := msg.ACK("AA", "")

	fmt.Println(lastName.Value, firstName.Value, val, mtype, et, trig, cid, ver, len(dg1s), len(jsonStr), len(ack))
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

### Message Access

```go
msg.Segment("PID")                   // first matching segment
msg.SegmentsByName("OBX")            // all matching segments
msg.Get("PID-5-1")                   // terser-style path access
msg.Get("OBX(1)-5")                  // segment repetition (0-based)
msg.MessageType()                    // ("ADT", "A01")
msg.ControlID()                      // "MSG001"
msg.Version()                        // "2.5.1"
msg.ACK("AA", "")                    // generate ACK response
msg.ToJSON()                         // JSON serialization
```

### MLLP

```go
medparse.IsMLLPFramed(data)          // detect MLLP framing
medparse.StripMLLP(raw)              // strip framing (Parse does this automatically)
```

## License

MIT
