package medparse

import (
	"time"
)

// ACK generates an ACK response message for this message.
//
// code is the acknowledgment code: "AA" (accept), "AE" (error), "AR" (reject).
// text is an optional text message for MSA-3.
func (m *Message) ACK(code, text string) (string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", err
	}

	fieldVal := func(idx int, fallback string) string {
		f, err := msh.Field(idx)
		if err != nil {
			return fallback
		}
		return f.Value
	}

	sep := fieldVal(1, "|")
	encChars := fieldVal(2, `^~\&`)
	sendApp := fieldVal(3, "")
	sendFac := fieldVal(4, "")
	recvApp := fieldVal(5, "")
	recvFac := fieldVal(6, "")
	controlID := fieldVal(10, "")
	version := fieldVal(12, "2.5")

	now := time.Now().Format("20060102150405")
	ackControlID := "ACK" + controlID

	// Build ACK: swap sender/receiver.
	ack := "MSH" + sep + encChars +
		sep + recvApp + sep + recvFac +
		sep + sendApp + sep + sendFac +
		sep + now + sep + sep + "ACK" +
		sep + ackControlID + sep + "P" + sep + version +
		"\rMSA" + sep + code + sep + controlID + sep + text

	return ack, nil
}
