package medparse

import "strings"

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

	// Split the raw input into individual messages at each MSH boundary.
	messageParts := splitIntoMessages(raw)

	results := make([]*Message, 0, len(messageParts))
	for _, msgStr := range messageParts {
		msgStr = strings.TrimSpace(msgStr)
		if len(msgStr) == 0 || !strings.HasPrefix(msgStr, "MSH") {
			continue
		}
		msg, err := Parse(msgStr)
		if err != nil {
			return nil, &ParseError{Msg: "error parsing message in batch: " + err.Error()}
		}
		results = append(results, msg)
	}

	return results, nil
}

// splitIntoMessages splits raw input into individual message strings at MSH boundaries.
// Strips FHS, BHS, BTS, FTS header/trailer lines.
func splitIntoMessages(raw string) []string {
	var messages []string
	var current strings.Builder

	// Split on any line ending.
	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\r' || r == '\n'
	})

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		segType := line
		if len(segType) > 3 {
			segType = line[:3]
		}

		switch segType {
		case "FHS", "BHS", "BTS", "FTS":
			// Skip batch/file header and trailer segments.
			continue
		case "MSH":
			// Start of a new message — flush the current one.
			if current.Len() > 0 {
				messages = append(messages, current.String())
				current.Reset()
			}
			current.WriteString(line)
		default:
			// Append to current message.
			if current.Len() > 0 {
				current.WriteByte('\r')
			}
			current.WriteString(line)
		}
	}

	// Flush the last message.
	if current.Len() > 0 {
		messages = append(messages, current.String())
	}

	return messages
}
