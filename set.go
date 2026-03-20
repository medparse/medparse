package medparse

import (
	"strconv"
	"strings"
)

// Set modifies a value in the message using terser-style path notation.
//
// Path format: SEGMENT-FIELD[-COMPONENT[-SUBCOMPONENT]]
//
// If the target index is beyond the current length, empty fields/components
// are auto-created to reach the specified position.
//
// Examples:
//
//	msg.Set("PID-5-1", "SMITH")   // modify last name
//	msg.Set("PID-8", "F")         // set gender
func (m *Message) Set(path, value string) error {
	parts := strings.Split(path, "-")
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

	// Parse field index.
	fieldIdx, err := strconv.Atoi(parts[1])
	if err != nil {
		return &ParseError{Msg: "invalid field index: '" + parts[1] + "'"}
	}
	if fieldIdx < 1 {
		return &IndexError{Type: "field", Index: fieldIdx, Max: len(seg.Fields)}
	}

	// Auto-extend fields if needed.
	for len(seg.Fields) < fieldIdx {
		seg.Fields = append(seg.Fields, emptyField())
	}
	field := &seg.Fields[fieldIdx-1]

	// Field-level set.
	if len(parts) == 2 {
		field.Value = value
		field.Components = []Component{{Value: value, SubComponents: []string{value}}}
		field.Repetitions = nil
		return nil
	}

	// Parse component index.
	compIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return &ParseError{Msg: "invalid component index: '" + parts[2] + "'"}
	}
	if compIdx < 1 {
		return &IndexError{Type: "component", Index: compIdx, Max: len(field.Components)}
	}

	// Auto-extend components if needed.
	for len(field.Components) < compIdx {
		field.Components = append(field.Components, emptyComponent())
	}
	comp := &field.Components[compIdx-1]

	// Component-level set.
	if len(parts) == 3 {
		comp.Value = value
		comp.SubComponents = []string{value}
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
	rebuildFieldValue(field, m.Enc)
	return nil
}

// rebuildFieldValue reconstructs the field's Value from its components.
func rebuildFieldValue(f *Field, enc EncodingChars) {
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
