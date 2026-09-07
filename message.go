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
	"time"
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

// Encode re-serializes the component using the given encoding characters.
func (c *Component) Encode(enc EncodingChars) string {
	if len(c.SubComponents) > 1 {
		return strings.Join(c.SubComponents, string(enc.SubComponentSep))
	}
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

// Encode re-serializes the field back to HL7 format using the given encoding characters.
func (f *Field) Encode(enc EncodingChars) string {
	if len(f.Repetitions) > 1 {
		parts := make([]string, len(f.Repetitions))
		for i := range f.Repetitions {
			parts[i] = f.Repetitions[i].Encode(enc)
		}
		return strings.Join(parts, string(enc.RepetitionSep))
	}
	if len(f.Components) > 1 {
		parts := make([]string, len(f.Components))
		for i := range f.Components {
			parts[i] = f.Components[i].Encode(enc)
		}
		return strings.Join(parts, string(enc.ComponentSep))
	}
	if len(f.Components) == 1 && len(f.Components[0].SubComponents) > 1 {
		return f.Components[0].Encode(enc)
	}
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

// Component returns a component at 1-based field and component indices.
func (s *Segment) Component(fieldIdx, compIdx int) (string, error) {
	f, err := s.Field(fieldIdx)
	if err != nil {
		return "", err
	}
	c, err := f.Component(compIdx)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

// AddField appends a field value to the segment using default encoding.
func (s *Segment) AddField(val string) {
	enc := DefaultEncodingChars()
	s.Fields = append(s.Fields, parseField(val, &enc))
}

// SetField sets or overwrites the field at a 1-based index, auto-extending if necessary.
func (s *Segment) SetField(index int, val string) error {
	if index < 1 {
		return &IndexError{Type: "field", Index: index, Max: len(s.Fields)}
	}
	for len(s.Fields) < index {
		s.Fields = append(s.Fields, emptyField())
	}
	enc := DefaultEncodingChars()
	s.Fields[index-1] = parseField(val, &enc)
	return nil
}

// ClearField sets the field at 1-based index to an empty field.
func (s *Segment) ClearField(index int) error {
	if index < 1 || index > len(s.Fields) {
		return &IndexError{Type: "field", Index: index, Max: len(s.Fields)}
	}
	s.Fields[index-1] = emptyField()
	return nil
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
		b.WriteString(s.Fields[i].Encode(enc))
	}

	return b.String()
}

// Message represents a parsed HL7v2 message.
type Message struct {
	Raw      string        `json:"-"`
	Segments []Segment     `json:"segments"`
	Enc      EncodingChars `json:"-"`
}

// NewMessage creates a new HL7v2 message with an initialized MSH segment.
// If version is omitted, "2.5.1" is used.
func NewMessage(eventType, triggerEvent string, version ...string) *Message {
	ver := "2.5.1"
	if len(version) > 0 && version[0] != "" {
		ver = version[0]
	}
	enc := DefaultEncodingChars()
	msgType := eventType
	if triggerEvent != "" {
		msgType = eventType + "^" + triggerEvent + "^" + eventType + "_" + triggerEvent
	}
	now := time.Now().Format("20060102150405")
	controlID := "MSG" + now

	msg := &Message{
		Enc: enc,
	}

	msh := msg.AddSegment("MSH")
	// MSH-1
	msh.Fields = append(msh.Fields, Field{
		Value: string(enc.FieldSep),
		Components: []Component{{
			Value:         string(enc.FieldSep),
			SubComponents: []string{string(enc.FieldSep)},
		}},
	})
	// MSH-2
	encStr := string([]byte{enc.ComponentSep, enc.RepetitionSep, enc.EscapeChar, enc.SubComponentSep})
	msh.Fields = append(msh.Fields, Field{
		Value: encStr,
		Components: []Component{{
			Value:         encStr,
			SubComponents: []string{encStr},
		}},
	})

	// MSH-3 through MSH-12:
	// 3: SendingApp, 4: SendingFacility, 5: ReceivingApp, 6: ReceivingFacility
	// 7: DateTime, 8: Security, 9: MessageType, 10: ControlID, 11: ProcessingID, 12: Version
	mshDefaults := []string{"", "", "", "", now, "", msgType, controlID, "P", ver}
	for _, val := range mshDefaults {
		msh.Fields = append(msh.Fields, parseField(val, &enc))
	}

	return msg
}

// Clone creates a deep copy of the message.
func (m *Message) Clone() *Message {
	clone := &Message{
		Raw:      m.Raw,
		Enc:      m.Enc,
		Segments: make([]Segment, len(m.Segments)),
	}
	for i, seg := range m.Segments {
		clone.Segments[i] = Segment{
			Name:   seg.Name,
			Fields: make([]Field, len(seg.Fields)),
		}
		for j, f := range seg.Fields {
			clone.Segments[i].Fields[j] = cloneField(&f)
		}
	}
	return clone
}

func cloneField(f *Field) Field {
	cf := Field{
		Value: f.Value,
	}
	if len(f.Components) > 0 {
		cf.Components = make([]Component, len(f.Components))
		for k, c := range f.Components {
			cf.Components[k] = Component{
				Value:         c.Value,
				SubComponents: append([]string(nil), c.SubComponents...),
			}
		}
	}
	if len(f.Repetitions) > 0 {
		cf.Repetitions = make([]Field, len(f.Repetitions))
		for r, rep := range f.Repetitions {
			cf.Repetitions[r] = cloneField(&rep)
		}
	}
	return cf
}

// AddSegment appends a new segment with the given name and field values.
// Returns a pointer to the added segment.
func (m *Message) AddSegment(name string, fields ...string) *Segment {
	seg := Segment{Name: name}
	if len(fields) > 0 {
		seg.Fields = make([]Field, len(fields))
		for i, fVal := range fields {
			seg.Fields[i] = parseField(fVal, &m.Enc)
		}
	}
	m.Segments = append(m.Segments, seg)
	return &m.Segments[len(m.Segments)-1]
}

// AddSegmentStruct appends an existing Segment struct to the message.
func (m *Message) AddSegmentStruct(seg Segment) {
	m.Segments = append(m.Segments, seg)
}

// InsertSegment inserts a segment at the specified 0-based index.
func (m *Message) InsertSegment(index int, seg Segment) error {
	if index < 0 || index > len(m.Segments) {
		return &IndexError{Type: "segment", Index: index, Max: len(m.Segments)}
	}
	m.Segments = append(m.Segments, Segment{})
	copy(m.Segments[index+1:], m.Segments[index:])
	m.Segments[index] = seg
	return nil
}

// RemoveSegment removes the segment at the specified 0-based index.
func (m *Message) RemoveSegment(index int) error {
	if index < 0 || index >= len(m.Segments) {
		return &IndexError{Type: "segment", Index: index, Max: len(m.Segments) - 1}
	}
	m.Segments = append(m.Segments[:index], m.Segments[index+1:]...)
	return nil
}

// RemoveSegmentsByName removes all segments with the specified name.
// Returns the count of removed segments.
func (m *Message) RemoveSegmentsByName(name string) int {
	var kept []Segment
	count := 0
	for _, seg := range m.Segments {
		if seg.Name == name {
			count++
		} else {
			kept = append(kept, seg)
		}
	}
	m.Segments = kept
	return count
}

// SegmentsByNamePtr returns pointers to all segments matching the given name,
// allowing in-place modification of segments.
func (m *Message) SegmentsByNamePtr(name string) []*Segment {
	var result []*Segment
	for i := range m.Segments {
		if m.Segments[i].Name == name {
			result = append(result, &m.Segments[i])
		}
	}
	return result
}

// AllSegments returns all segments in the message.
func (m *Message) AllSegments() []Segment {
	return m.Segments
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
