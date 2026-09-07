package medparse

import (
	"strconv"
	"strings"
)

// Set modifies a value in the message using terser-style path notation.
//
// Path format: SEGMENT-FIELD[-COMPONENT[-SUBCOMPONENT]]
//
// Both dash (-) and dot (.) notations are supported (e.g. "PID-5-1" or "PID-5.1" or "PID.5.1").
//
// Field repetitions can be specified with parentheses:
//   - msg.Set("PID-3(1)-1", "DEA123")
//
// If the target index is beyond the current length, empty fields/components
// are auto-created to reach the specified position.
//
// Examples:
//
//	msg.Set("PID-5-1", "SMITH")     // modify last name
//	msg.Set("PID-5.1", "SMITH")     // dot notation
//	msg.Set("PID-8", "F")           // set gender
//	msg.Set("PID-3(1)-1", "DEA123") // set second repetition of PID-3
func (m *Message) Set(path, value string) error {
	parts := splitTerserPath(path)
	if len(parts) < 2 {
		return &ParseError{Msg: "set path must include at least SEGMENT-FIELD"}
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
		return &KeyError{Name: segName}
	}

	if segRep >= len(matching) {
		return &IndexError{
			Type:  "segment repetition",
			Index: segRep,
			Max:   len(matching),
		}
	}

	seg := matching[segRep]

	// Parse field index and optional field repetition index.
	fieldIdx, fieldRep, err := parseFieldRef(parts[1])
	if err != nil {
		return err
	}
	if fieldIdx < 1 {
		return &IndexError{Type: "field", Index: fieldIdx, Max: len(seg.Fields)}
	}

	// Auto-extend fields if needed.
	for len(seg.Fields) < fieldIdx {
		seg.Fields = append(seg.Fields, emptyField())
	}
	field := &seg.Fields[fieldIdx-1]

	// Determine target field (either base field or a specific repetition).
	targetField := field
	if fieldRep >= 0 {
		if len(field.Repetitions) == 0 {
			// Initialize repetitions with the existing field as repetition 0.
			field.Repetitions = []Field{cloneField(field)}
		}
		for len(field.Repetitions) <= fieldRep {
			field.Repetitions = append(field.Repetitions, emptyField())
		}
		targetField = &field.Repetitions[fieldRep]
	} else if len(field.Repetitions) > 0 {
		// No repetition specified, but repetitions exist — update repetition 0.
		targetField = &field.Repetitions[0]
	}

	// Field-level set.
	if len(parts) == 2 {
		targetField.Value = value
		targetField.Components = []Component{{Value: value, SubComponents: []string{value}}}
		if fieldRep < 0 {
			field.Repetitions = nil
		}
		rebuildFieldValue(field, m.Enc)
		return nil
	}

	// Parse component index.
	compIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return &ParseError{Msg: "invalid component index: '" + parts[2] + "'"}
	}
	if compIdx < 1 {
		return &IndexError{Type: "component", Index: compIdx, Max: len(targetField.Components)}
	}

	// Auto-extend components if needed.
	for len(targetField.Components) < compIdx {
		targetField.Components = append(targetField.Components, emptyComponent())
	}
	comp := &targetField.Components[compIdx-1]

	// Component-level set.
	if len(parts) == 3 {
		comp.Value = value
		comp.SubComponents = []string{value}
		rebuildSingleFieldValue(targetField, m.Enc)
		rebuildFieldValue(field, m.Enc)
		return nil
	}

	// Parse sub-component index.
	subIdx, err := strconv.Atoi(parts[3])
	if err != nil {
		return &ParseError{Msg: "invalid sub-component index: '" + parts[3] + "'"}
	}
	if subIdx < 1 {
		return &IndexError{Type: "sub-component", Index: subIdx, Max: len(comp.SubComponents)}
	}

	// Auto-extend sub-components if needed.
	for len(comp.SubComponents) < subIdx {
		comp.SubComponents = append(comp.SubComponents, "")
	}

	comp.SubComponents[subIdx-1] = value
	rebuildComponentValue(comp, m.Enc)
	rebuildSingleFieldValue(targetField, m.Enc)
	rebuildFieldValue(field, m.Enc)
	return nil
}

// rebuildFieldValue reconstructs the field's Value from its components or repetitions.
func rebuildFieldValue(f *Field, enc EncodingChars) {
	if len(f.Repetitions) > 0 {
		sep := string(enc.RepetitionSep)
		parts := make([]string, len(f.Repetitions))
		for i := range f.Repetitions {
			rebuildSingleFieldValue(&f.Repetitions[i], enc)
			parts[i] = f.Repetitions[i].Value
		}
		f.Value = strings.Join(parts, sep)
		f.Components = f.Repetitions[0].Components
		return
	}
	rebuildSingleFieldValue(f, enc)
}

// rebuildSingleFieldValue reconstructs a single field value from its components.
func rebuildSingleFieldValue(f *Field, enc EncodingChars) {
	sep := string(enc.ComponentSep)
	parts := make([]string, len(f.Components))
	for i, c := range f.Components {
		parts[i] = c.Value
	}
	f.Value = strings.Join(parts, sep)
}

// rebuildComponentValue reconstructs the component's Value from its sub-components.
func rebuildComponentValue(c *Component, enc EncodingChars) {
	sep := string(enc.SubComponentSep)
	c.Value = strings.Join(c.SubComponents, sep)
}

// emptyField creates an empty field with a single empty component.
func emptyField() Field {
	return Field{
		Value:      "",
		Components: []Component{emptyComponent()},
	}
}

// emptyComponent creates an empty component.
func emptyComponent() Component {
	return Component{
		Value:         "",
		SubComponents: []string{""},
	}
}
