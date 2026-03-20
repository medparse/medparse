package medparse

// MLLP (Minimal Lower Layer Protocol) framing utilities.
//
// HL7 messages transmitted over TCP are wrapped in MLLP framing:
//   - Start: 0x0B (vertical tab / VT)
//   - End:   0x1C (file separator) optionally followed by 0x0D (carriage return)

const (
	mllpStart byte = 0x0B // VT (vertical tab)
	mllpEnd   byte = 0x1C // FS (file separator)
)

// IsMLLPFramed checks if raw bytes are wrapped in MLLP framing.
func IsMLLPFramed(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	last := data[len(data)-1]
	secondLast := data[len(data)-2]
	return data[0] == mllpStart &&
		(last == '\r' && secondLast == mllpEnd || last == mllpEnd)
}

// StripMLLP strips MLLP framing from raw data, returning the inner message.
// If the data is not MLLP-framed, returns it unchanged.
func StripMLLP(data string) string {
	b := []byte(data)
	if !IsMLLPFramed(b) {
		return data
	}

	start := 1 // skip 0x0B
	end := len(b) - 1
	if b[len(b)-1] == '\r' && b[len(b)-2] == mllpEnd {
		end = len(b) - 2
	}

	return data[start:end]
}
