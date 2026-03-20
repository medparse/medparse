package medparse_test

import (
	"fmt"

	"github.com/medparse/medparse"
)

func ExampleParse() {
	msg, err := medparse.Parse(
		"MSH|^~\\&|EPIC|HOSPITAL|RECV|FAC|20260318||ADT^A01|MSG001|P|2.5.1\r" +
			"PID|1||MRN12345^^^MRN||DOE^JANE^M||19850315|F\r" +
			"PV1|1|I|4EAST^401^1",
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(msg.Segments), "segments parsed")
	// Output: 3 segments parsed
}

func ExampleMessage_Get() {
	msg, _ := medparse.Parse(
		"MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|12345|P|2.5\r" +
			"PID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M",
	)

	lastName, _ := msg.Get("PID-5-1")
	firstName, _ := msg.Get("PID-5-2")
	msgType, _ := msg.Get("MSH-9-1")

	fmt.Println(lastName, firstName, msgType)
	// Output: DOE JOHN ADT
}

func ExampleMessage_Set() {
	msg, _ := medparse.Parse(
		"MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|12345|P|2.5\r" +
			"PID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M",
	)

	msg.Set("PID-5-1", "SMITH")
	msg.Set("PID-5-2", "JANE")

	lastName, _ := msg.Get("PID-5-1")
	firstName, _ := msg.Get("PID-5-2")

	fmt.Println(lastName, firstName)
	// Output: SMITH JANE
}

func ExampleMessage_ACK() {
	msg, _ := medparse.Parse(
		"MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|12345|P|2.5\r" +
			"PID|1||MRN",
	)

	ack, _ := msg.ACK("AA", "")
	// ACK starts with MSH and contains MSA with the original control ID.
	fmt.Println(ack[:3])
	// Output: MSH
}

func ExampleParseBatch() {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\rPID|1||MRN1\r" +
		"MSH|^~\\&|S|F|R|F|20230102||ADT^A01|2|P|2.5\rPID|1||MRN2"

	msgs, _ := medparse.ParseBatch(raw)
	fmt.Println(len(msgs), "messages")
	// Output: 2 messages
}

func ExampleParseTimestamp() {
	ts, _ := medparse.ParseTimestamp("20230315143022-0500")
	fmt.Println(ts.Year(), ts.Month(), ts.Day(), ts.Hour())
	// Output: 2023 March 15 14
}

func ExampleMessage_EachSegment() {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ORU^R01|1|P|2.5\r" +
		"PID|1||MRN\rOBR|1\r" +
		"OBX|1|NM|HR||72\r" +
		"OBX|2|NM|BP||120"

	msg, _ := medparse.Parse(raw)

	count := 0
	msg.EachSegment("OBX", func(i int, seg *medparse.Segment) error {
		count++
		return nil
	})

	fmt.Println(count, "OBX segments")
	// Output: 2 OBX segments
}

func ExampleMessage_Validate() {
	raw := "MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\r" +
		"EVN|A01\rPID|1||MRN\rPV1|1|I|4EAST"

	msg, _ := medparse.Parse(raw)

	err := msg.Validate()
	fmt.Println(err)
	// Output: <nil>
}

func ExampleMessage_String() {
	msg, _ := medparse.Parse(
		"MSH|^~\\&|SENDER|FAC|RECV|FAC|20230101||ADT^A01|12345|P|2.5\r" +
			"PID|1||MRN123^^^MRN||DOE^JOHN",
	)

	msg.Set("PID-5-1", "SMITH")

	// Re-serialize to HL7.
	raw := msg.String()
	fmt.Println(raw[:3]) // starts with MSH
	// Output: MSH
}
