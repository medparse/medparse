package medparse

import (
	"bufio"
	"bytes"
	"io"
)

// MLLP (Minimal Lower Layer Protocol) framing utilities.
//
// HL7 messages transmitted over TCP are wrapped in MLLP framing:
//   - Start: 0x0B (vertical tab / VT)
//   - End:   0x1C (file separator) followed by 0x0D (carriage return)

const (
	mllpStart byte = 0x0B // VT (vertical tab)
	mllpEnd   byte = 0x1C // FS (file separator)
	mllpCR    byte = 0x0D // CR (carriage return)
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

// WrapMLLP wraps message bytes in standard MLLP framing (<VT> + data + <FS><CR>).
func WrapMLLP(data []byte) []byte {
	out := make([]byte, len(data)+3)
	out[0] = mllpStart
	copy(out[1:], data)
	out[len(out)-2] = mllpEnd
	out[len(out)-1] = mllpCR
	return out
}

// WrapMLLPString wraps a message string in standard MLLP framing.
func WrapMLLPString(data string) string {
	return string(WrapMLLP([]byte(data)))
}

// ScanMLLP is a bufio.SplitFunc that extracts MLLP-framed HL7 messages from a stream.
// It strips framing bytes and handles optional trailing carriage return.
func ScanMLLP(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Find the start byte (0x0B).
	startIdx := bytes.IndexByte(data, mllpStart)
	if startIdx < 0 {
		if atEOF {
			// Discard trailing non-MLLP bytes.
			return len(data), nil, nil
		}
		// Request more data.
		return 0, nil, nil
	}

	// Advance past any noise before the start byte.
	if startIdx > 0 {
		return startIdx, nil, nil
	}

	// Find the closing 0x1C.
	endIdx := bytes.IndexByte(data[1:], mllpEnd)
	if endIdx < 0 {
		if atEOF {
			return 0, nil, &ParseError{Msg: "truncated MLLP frame: missing closing FS byte"}
		}
		return 0, nil, nil
	}

	endIdx += 1 // adjust because we sliced data[1:]
	advance = endIdx + 1
	if advance < len(data) && data[advance] == mllpCR {
		advance++
	}

	return advance, data[1:endIdx], nil
}

// MLLPReader reads MLLP-framed HL7v2 messages from an io.Reader (such as a net.Conn).
type MLLPReader struct {
	scanner *bufio.Scanner
}

// NewMLLPReader creates a new MLLPReader wrapping the provided io.Reader.
func NewMLLPReader(r io.Reader) *MLLPReader {
	scanner := bufio.NewScanner(r)
	scanner.Split(ScanMLLP)
	return &MLLPReader{scanner: scanner}
}

// ReadMessage reads the next MLLP-framed message from the stream.
// Returns io.EOF when the stream is exhausted.
func (r *MLLPReader) ReadMessage() ([]byte, error) {
	if r.scanner.Scan() {
		return r.scanner.Bytes(), nil
	}
	if err := r.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// ReadString reads the next MLLP-framed message as a string.
// Returns io.EOF when the stream is exhausted.
func (r *MLLPReader) ReadString() (string, error) {
	if r.scanner.Scan() {
		return r.scanner.Text(), nil
	}
	if err := r.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// MLLPWriter writes MLLP-framed HL7v2 messages to an io.Writer (such as a net.Conn).
type MLLPWriter struct {
	w io.Writer
}

// NewMLLPWriter creates a new MLLPWriter wrapping the provided io.Writer.
func NewMLLPWriter(w io.Writer) *MLLPWriter {
	return &MLLPWriter{w: w}
}

// WriteMessage writes an HL7v2 message wrapped in standard MLLP framing.
func (w *MLLPWriter) WriteMessage(data []byte) error {
	framed := WrapMLLP(data)
	_, err := w.w.Write(framed)
	return err
}

// WriteString writes an HL7v2 message string wrapped in standard MLLP framing.
func (w *MLLPWriter) WriteString(data string) error {
	return w.WriteMessage([]byte(data))
}
