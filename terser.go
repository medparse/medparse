package medparse

import (
	"strconv"
	"strings"
)

// Get returns a value from the message using terser-style path notation.
//
// Path format: SEGMENT-FIELD[-COMPONENT[-SUBCOMPONENT]]
//
// Both dash (-) and dot (.) notations are supported (e.g. "PID-5-1" or "PID-5.1" or "PID.5.1").
//
// Segment and field repetitions can be specified with parentheses:
//   - "OBX(1)-5"   → second OBX's field 5 (0-based)
//   - "PID-3(1)-1" → second repetition of PID-3's component 1 (0-based)
//
// Examples:
//
//	msg.Get("PID-5-1")      → "DOE"
//	msg.Get("PID-5.1")      → "DOE"
//	msg.Get("MSH-9-1")      → "ADT"
//	msg.Get("OBX(1)-5")     → second OBX's field 5
//	msg.Get("PID-3(1)-1")   → second PID-3 repetition's first component
func (m *Message) Get(path string) (string, error) {
	parts := splitTerserPath(path)
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

	fieldIdx, fieldRep, err := parseFieldRef(parts[1])
	if err != nil {
		return "", err
	}
	field, err := seg.Field(fieldIdx)
	if err != nil {
		return "", err
	}

	// Resolve field repetition if specified.
	targetField := field
	if fieldRep >= 0 {
		if len(field.Repetitions) > 0 {
			if fieldRep >= len(field.Repetitions) {
				return "", &IndexError{
					Type:  "field repetition",
					Index: fieldRep,
					Max:   len(field.Repetitions),
				}
			}
			targetField = &field.Repetitions[fieldRep]
		} else if fieldRep != 0 {
			return "", &IndexError{
				Type:  "field repetition",
				Index: fieldRep,
				Max:   1,
			}
		}
	}

	if len(parts) == 2 {
		return targetField.Value, nil
	}

	compIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", &ParseError{Msg: "invalid component index: '" + parts[2] + "'"}
	}
	comp, err := targetField.Component(compIdx)
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

// GetAll returns all values matching the path across segment and field repetitions.
//
// Examples:
//   - msg.GetAll("OBX-5")   → returns field 5 for each OBX segment
//   - msg.GetAll("PID-3-1") → returns component 1 for each repetition of PID-3
func (m *Message) GetAll(path string) ([]string, error) {
	parts := splitTerserPath(path)
	if len(parts) == 0 {
		return nil, &ParseError{Msg: "empty terser path"}
	}

	segName, segRep := parseSegmentRef(parts[0])

	var matching []*Segment
	for i := range m.Segments {
		if m.Segments[i].Name == segName {
			matching = append(matching, &m.Segments[i])
		}
	}

	if len(matching) == 0 {
		return nil, &KeyError{Name: segName}
	}

	// If a specific segment repetition was requested e.g. "OBX(0)", narrow to that one.
	if strings.Contains(parts[0], "(") {
		if segRep >= len(matching) {
			return nil, &IndexError{Type: "segment repetition", Index: segRep, Max: len(matching)}
		}
		matching = matching[segRep : segRep+1]
	}

	if len(parts) == 1 {
		res := make([]string, len(matching))
		for i, s := range matching {
			res[i] = s.Name
		}
		return res, nil
	}

	fieldIdx, fieldRep, err := parseFieldRef(parts[1])
	if err != nil {
		return nil, err
	}

	var results []string
	for _, seg := range matching {
		field, err := seg.Field(fieldIdx)
		if err != nil {
			continue
		}

		var targetFields []*Field
		if fieldRep >= 0 {
			if len(field.Repetitions) > 0 {
				if fieldRep < len(field.Repetitions) {
					targetFields = append(targetFields, &field.Repetitions[fieldRep])
				}
			} else if fieldRep == 0 {
				targetFields = append(targetFields, field)
			}
		} else {
			if len(field.Repetitions) > 0 {
				for r := range field.Repetitions {
					targetFields = append(targetFields, &field.Repetitions[r])
				}
			} else {
				targetFields = append(targetFields, field)
			}
		}

		for _, tf := range targetFields {
			if len(parts) == 2 {
				results = append(results, tf.Value)
				continue
			}

			compIdx, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, &ParseError{Msg: "invalid component index: '" + parts[2] + "'"}
			}
			comp, err := tf.Component(compIdx)
			if err != nil {
				continue
			}

			if len(parts) == 3 {
				results = append(results, comp.Value)
				continue
			}

			subIdx, err := strconv.Atoi(parts[3])
			if err != nil {
				return nil, &ParseError{Msg: "invalid sub-component index: '" + parts[3] + "'"}
			}
			sub, err := comp.SubComponent(subIdx)
			if err != nil {
				continue
			}
			results = append(results, sub)
		}
	}

	return results, nil
}

// splitTerserPath splits a path on '-' or '.' delimiters.
func splitTerserPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '-' || path[i] == '.' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
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

// parseFieldRef parses a field reference like "5" or "3(1)" into (fieldIdx, repIdx).
// If no repetition is specified, repIdx is -1.
func parseFieldRef(s string) (int, int, error) {
	parenStart := strings.IndexByte(s, '(')
	parenEnd := strings.IndexByte(s, ')')

	if parenStart >= 0 && parenEnd > parenStart {
		fieldStr := s[:parenStart]
		fIdx, err := strconv.Atoi(fieldStr)
		if err != nil {
			return 0, -1, &ParseError{Msg: "invalid field index: '" + s + "'"}
		}
		repIdx, err := strconv.Atoi(s[parenStart+1 : parenEnd])
		if err != nil {
			return 0, -1, &ParseError{Msg: "invalid field repetition index: '" + s + "'"}
		}
		return fIdx, repIdx, nil
	}

	fIdx, err := strconv.Atoi(s)
	if err != nil {
		return 0, -1, &ParseError{Msg: "invalid field index: '" + s + "'"}
	}
	return fIdx, -1, nil
}
