// Package medparse provides a high-performance HL7v2 message parser for Go.
//
// It supports the full HL7v2 hierarchy (Message → Segment → Field → Component →
// Sub-component), MLLP framing, escape sequences, terser-style path access,
// batch parsing, timestamp parsing, and ACK generation.
//
// Zero external dependencies — standard library only.
package medparse

import (
	"encoding/json"
	"strings"
)

// EncodingChars holds the HL7v2 encoding characters extracted from the MSH segment.
type EncodingChars struct {
	FieldSep        byte
	ComponentSep    byte
	RepetitionSep   byte
	EscapeChar      byte
	SubComponentSep byte
}

// DefaultEncodingChars returns the standard HL7v2 encoding characters.
func DefaultEncodingChars() EncodingChars {
	return EncodingChars{
		FieldSep:        '|',
		ComponentSep:    '^',
		RepetitionSep:   '~',
		EscapeChar:      '\\',
		SubComponentSep: '&',
	}
}

// Component represents a single component within an HL7v2 field.
// Components may contain sub-components separated by '&'.
type Component struct {
	Value         string   `json:"value"`
	SubComponents []string `json:"sub_components"`
}

// SubComponent returns the sub-component at a 1-based index.
func (c *Component) SubComponent(index int) (string, error) {
	if index < 1 || index > len(c.SubComponents) {
		return "", &IndexError{Type: "sub-component", Index: index, Max: len(c.SubComponents)}
	}
	return c.SubComponents[index-1], nil
}

// String returns the component value.
func (c *Component) String() string {
	return c.Value
}

// Field represents a single field within an HL7v2 segment.
// Fields may contain components (separated by '^') and repetitions (separated by '~').
type Field struct {
	Value       string      `json:"value"`
	Components  []Component `json:"components"`
	Repetitions []Field     `json:"repetitions,omitempty"`
}

// Component returns the component at a 1-based index.
func (f *Field) Component(index int) (*Component, error) {
	if index < 1 || index > len(f.Components) {
		return nil, &IndexError{Type: "component", Index: index, Max: len(f.Components)}
	}
	return &f.Components[index-1], nil
}

// String returns the field value.
func (f *Field) String() string {
	return f.Value
}

// Segment represents a single segment within an HL7v2 message (e.g. MSH, PID, OBX).
type Segment struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field returns the field at a 1-based index.
func (s *Segment) Field(index int) (*Field, error) {
	if index < 1 || index > len(s.Fields) {
		return nil, &IndexError{Type: "field", Index: index, Max: len(s.Fields)}
	}
	return &s.Fields[index-1], nil
}

// String re-serializes the segment back to HL7 pipe-delimited format using
// the default encoding characters.
func (s *Segment) String() string {
	return s.Encode(DefaultEncodingChars())
}

// Encode re-serializes the segment using the given encoding characters.
func (s *Segment) Encode(enc EncodingChars) string {
	if len(s.Fields) == 0 {
		return s.Name
	}

	sep := string(enc.FieldSep)
	isMSH := s.Name == "MSH"

	var b strings.Builder
	b.WriteString(s.Name)
	b.WriteByte(enc.FieldSep)

	startIdx := 0
	if isMSH {
		// MSH-1 is the field separator (already written above).
		// MSH-2 is the encoding characters.
		startIdx = 1
	}

	for i := startIdx; i < len(s.Fields); i++ {
		if i > startIdx {
			b.WriteString(sep)
		}
		b.WriteString(s.Fields[i].Value)
	}

	return b.String()
}

// Message represents a parsed HL7v2 message.
type Message struct {
	Raw      string        `json:"-"`
	Segments []Segment     `json:"segments"`
	Enc      EncodingChars `json:"-"`
}

// Segment returns the first segment matching the given name.
func (m *Message) Segment(name string) (*Segment, error) {
	for i := range m.Segments {
		if m.Segments[i].Name == name {
			return &m.Segments[i], nil
		}
	}
	return nil, &KeyError{Name: name}
}

// SegmentsByName returns all segments matching the given name.
func (m *Message) SegmentsByName(name string) []Segment {
	var result []Segment
	for _, seg := range m.Segments {
		if seg.Name == name {
			result = append(result, seg)
		}
	}
	return result
}

// EachSegment iterates over all segments matching the given name.
// The callback receives the 0-based repetition index and a pointer to the segment.
// Return a non-nil error from the callback to stop iteration early.
func (m *Message) EachSegment(name string, fn func(index int, seg *Segment) error) error {
	idx := 0
	for i := range m.Segments {
		if m.Segments[i].Name == name {
			if err := fn(idx, &m.Segments[i]); err != nil {
				return err
			}
			idx++
		}
	}
	return nil
}

// MessageType returns the message type from MSH-9, e.g. ("ADT", "A01").
func (m *Message) MessageType() (string, string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", "", err
	}
	f, err := msh.Field(9)
	if err != nil {
		return "", "", err
	}
	eventType := ""
	trigger := ""
	if len(f.Components) > 0 {
		eventType = f.Components[0].Value
	}
	if len(f.Components) >= 2 {
		trigger = f.Components[1].Value
	}
	return eventType, trigger, nil
}

// ControlID returns the message control ID from MSH-10.
func (m *Message) ControlID() (string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", err
	}
	f, err := msh.Field(10)
	if err != nil {
		return "", err
	}
	return f.Value, nil
}

// Version returns the HL7 version from MSH-12.
func (m *Message) Version() (string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", err
	}
	f, err := msh.Field(12)
	if err != nil {
		return "", err
	}
	return f.Value, nil
}

// SendingApplication returns the sending application from MSH-3.
func (m *Message) SendingApplication() (string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", err
	}
	f, err := msh.Field(3)
	if err != nil {
		return "", err
	}
	return f.Value, nil
}

// SendingFacility returns the sending facility from MSH-4.
func (m *Message) SendingFacility() (string, error) {
	msh, err := m.Segment("MSH")
	if err != nil {
		return "", err
	}
	f, err := msh.Field(4)
	if err != nil {
		return "", err
	}
	return f.Value, nil
}

// ToJSON serializes the message to a JSON string.
func (m *Message) ToJSON() (string, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// String re-serializes the message back to HL7 pipe-delimited format
// using the stored encoding characters. Segments are separated by \r.
func (m *Message) String() string {
	parts := make([]string, len(m.Segments))
	for i := range m.Segments {
		parts[i] = m.Segments[i].Encode(m.Enc)
	}
	return strings.Join(parts, "\r")
}
