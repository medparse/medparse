package medparse

import (
	"time"
)

// ACK generates an ACK response message string for this message.
//
// code is the acknowledgment code: "AA" (accept), "AE" (error), "AR" (reject).
// text is an optional text message for MSA-3.
func (m *Message) ACK(code, text string) (string, error) {
	ack, err := m.BuildACK(code, text)
	if err != nil {
		return "", err
	}
	return ack.String(), nil
}

// BuildACK creates an ACK response *Message for this message.
//
// Swaps sender and receiver applications and facilities, preserves version
// and processing ID, and generates an MSA segment referencing this message's control ID.
func (m *Message) BuildACK(code, text string) (*Message, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return nil, err
	}

	fieldVal := func(idx int, fallback string) string {
		f, err := msh.Field(idx)
		if err != nil {
			return fallback
		}
		return f.Value
	}

	sendApp := fieldVal(3, "")
	sendFac := fieldVal(4, "")
	recvApp := fieldVal(5, "")
	recvFac := fieldVal(6, "")
	controlID := fieldVal(10, "")
	procID := fieldVal(11, "P")
	version := fieldVal(12, "2.5")

	now := time.Now().Format("20060102150405")
	ackControlID := "ACK" + controlID

	ack := &Message{
		Enc: m.Enc,
	}

	mshSeg := ack.AddSegment("MSH")
	// MSH-1
	mshSeg.Fields = append(mshSeg.Fields, Field{
		Value: string(m.Enc.FieldSep),
		Components: []Component{{
			Value:         string(m.Enc.FieldSep),
			SubComponents: []string{string(m.Enc.FieldSep)},
		}},
	})
	// MSH-2
	encStr := string([]byte{m.Enc.ComponentSep, m.Enc.RepetitionSep, m.Enc.EscapeChar, m.Enc.SubComponentSep})
	mshSeg.Fields = append(mshSeg.Fields, Field{
		Value: encStr,
		Components: []Component{{
			Value:         encStr,
			SubComponents: []string{encStr},
		}},
	})

	mshFields := []string{recvApp, recvFac, sendApp, sendFac, now, "", "ACK", ackControlID, procID, version}
	for _, val := range mshFields {
		mshSeg.Fields = append(mshSeg.Fields, parseField(val, &m.Enc))
	}

	// Add MSA segment (MSA-1: code, MSA-2: controlID, MSA-3: text)
	ack.AddSegment("MSA", code, controlID, text)

	return ack, nil
}

// AddErrorSegment adds an ERR segment to the message for structured error reporting in ACKs.
//
// severity is typically "E" (error), "W" (warning), or "I" (information).
// location is the HL7 path or location (e.g. "PID-3-1").
func (m *Message) AddErrorSegment(severity, code, text, location string) *Segment {
	return m.AddSegment("ERR", location, code, severity, text)
}
