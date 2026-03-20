package medparse

import (
	"strconv"
	"strings"
)

// Get returns a value from the message using terser-style path notation.
//
// Path format: SEGMENT-FIELD[-COMPONENT[-SUBCOMPONENT]]
//
// Segment repetition can be specified with parentheses: SEGMENT(n) where n is 0-based.
//
// Examples:
//
//	msg.Get("PID-5-1")     → "DOE"
//	msg.Get("MSH-9-1")     → "ADT"
//	msg.Get("OBX(1)-5")    → second OBX's field 5
func (m *Message) Get(path string) (string, error) {
	parts := strings.Split(path, "-")
	if len(parts) == 0 {
		return "", &ParseError{Msg: "empty terser path"}
	}

	// Parse segment name and optional repetition index.
	segName, segRep := parseSegmentRef(parts[0])

	// Find matching segments.
	var matching []*Segment
	for i := range m.Segments {
		if m.Segments[i].Name == segName {
			matching = append(matching, &m.Segments[i])
		}
	}

	if len(matching) == 0 {
		return "", &KeyError{Name: segName}
	}

	if segRep >= len(matching) {
		return "", &IndexError{
			Type:  "segment repetition",
			Index: segRep,
			Max:   len(matching),
		}
	}

	seg := matching[segRep]

	// If no field specified, return segment name.
	if len(parts) == 1 {
		return seg.Name, nil
	}

	fieldIdx, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", &ParseError{Msg: "invalid field index: '" + parts[1] + "'"}
	}
	field, err := seg.Field(fieldIdx)
	if err != nil {
		return "", err
	}

	if len(parts) == 2 {
		return field.Value, nil
	}

	compIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", &ParseError{Msg: "invalid component index: '" + parts[2] + "'"}
	}
	comp, err := field.Component(compIdx)
	if err != nil {
		return "", err
	}

	if len(parts) == 3 {
		return comp.Value, nil
	}

	subIdx, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", &ParseError{Msg: "invalid sub-component index: '" + parts[3] + "'"}
	}
	return comp.SubComponent(subIdx)
}

// parseSegmentRef parses a segment reference like "PID" or "OBX(1)" into (name, repetition_index).
func parseSegmentRef(s string) (string, int) {
	parenStart := strings.IndexByte(s, '(')
	parenEnd := strings.IndexByte(s, ')')

	if parenStart >= 0 && parenEnd > parenStart {
		name := s[:parenStart]
		idx, err := strconv.Atoi(s[parenStart+1 : parenEnd])
		if err != nil {
			return s, 0
		}
		return name, idx
	}

	return s, 0
}
