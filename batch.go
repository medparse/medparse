package medparse

import (
	"bufio"
	"io"
	"strings"
)

// ParseBatch parses a batch of HL7v2 messages from a raw string.
//
// Handles:
//   - FHS/BHS/BTS/FTS wrapped files (batch headers/trailers are stripped)
//   - Multiple messages separated by MSH segment headers
//
// Returns a slice of parsed Message objects.
func ParseBatch(raw string) ([]*Message, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}

	return ParseBatchReader(strings.NewReader(raw))
}

// ParseBatchReader parses all HL7v2 messages from an io.Reader stream.
// Supports large batches with low memory overhead.
func ParseBatchReader(r io.Reader) ([]*Message, error) {
	scanner := NewBatchScanner(r)
	var results []*Message
	for scanner.Scan() {
		results = append(results, scanner.Message())
	}
	if err := scanner.Err(); err != nil {
		return nil, &ParseError{Msg: "error parsing message in batch: " + err.Error()}
	}
	return results, nil
}

// BatchScanner scans messages sequentially from an io.Reader stream.
// It handles FHS/BHS/BTS/FTS batch wrappers and splits on MSH boundaries.
type BatchScanner struct {
	scanner *bufio.Scanner
	current strings.Builder
	msg     *Message
	err     error
	done    bool
}

// NewBatchScanner creates a new BatchScanner for the provided io.Reader.
func NewBatchScanner(r io.Reader) *BatchScanner {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanHL7Lines)
	return &BatchScanner{scanner: scanner}
}

// Scan advances to the next message in the batch.
// Returns false when the batch is finished or an error occurs.
func (s *BatchScanner) Scan() bool {
	if s.done || s.err != nil {
		return false
	}

	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if len(line) == 0 {
			continue
		}

		segType := line
		if len(segType) > 3 {
			segType = line[:3]
		}

		switch segType {
		case "FHS", "BHS", "BTS", "FTS":
			// Skip batch and file headers/trailers.
			continue
		case "MSH":
			if s.current.Len() > 0 {
				msg, err := Parse(s.current.String())
				if err != nil {
					s.err = err
					return false
				}
				s.msg = msg
				s.current.Reset()
				s.current.WriteString(line)
				return true
			}
			s.current.WriteString(line)
		default:
			if s.current.Len() > 0 {
				s.current.WriteByte('\r')
			}
			s.current.WriteString(line)
		}
	}

	if err := s.scanner.Err(); err != nil {
		s.err = err
		return false
	}

	if s.current.Len() > 0 {
		msg, err := Parse(s.current.String())
		s.current.Reset()
		s.done = true
		if err != nil {
			s.err = err
			return false
		}
		s.msg = msg
		return true
	}

	s.done = true
	return false
}

// Message returns the most recently scanned message.
func (s *BatchScanner) Message() *Message {
	return s.msg
}

// Err returns the first non-EOF error encountered by the scanner.
func (s *BatchScanner) Err() error {
	return s.err
}

// scanHL7Lines is a bufio.SplitFunc that splits lines on \r, \n, or \r\n.
func scanHL7Lines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i := 0; i < len(data); i++ {
		if data[i] == '\r' {
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
			return i + 1, data[:i], nil
		}
		if data[i] == '\n' {
			return i + 1, data[:i], nil
		}
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}
