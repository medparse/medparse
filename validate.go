package medparse

// requiredSegments defines the required segments for common HL7 message types.
// Key is "EventType_TriggerEvent" (e.g. "ADT_A01").
var requiredSegments = map[string][]string{
	// ADT messages
	"ADT_A01": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A02": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A03": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A04": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A05": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A06": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A07": {"MSH", "EVN", "PID", "PV1"},
	"ADT_A08": {"MSH", "EVN", "PID", "PV1"},

	// Order messages
	"ORM_O01": {"MSH", "PID", "ORC"},
	"OML_O21": {"MSH", "PID", "ORC", "OBR"},

	// Result messages
	"ORU_R01": {"MSH", "PID", "OBR", "OBX"},

	// Scheduling
	"SIU_S12": {"MSH", "SCH", "PID"},

	// Master file
	"MFN_M01": {"MSH", "MFI"},

	// Acknowledgment
	"ACK": {"MSH", "MSA"},
}

// Validate checks that the message contains all required segments for its
// message type. Returns nil if the message is valid or if the message type
// has no known validation rules.
//
// This is opt-in validation — Parse does not call this automatically.
func (m *Message) Validate() error {
	eventType, trigger, err := m.MessageType()
	if err != nil {
		return err
	}

	// Try specific match first (e.g. "ADT_A01"), then generic (e.g. "ACK").
	key := eventType + "_" + trigger
	required, ok := requiredSegments[key]
	if !ok {
		required, ok = requiredSegments[eventType]
		if !ok {
			return nil // no rules for this message type
		}
		key = eventType
	}

	// Build set of segment names present in the message.
	present := make(map[string]bool, len(m.Segments))
	for _, seg := range m.Segments {
		present[seg.Name] = true
	}

	// Check for missing required segments.
	var missing []string
	for _, name := range required {
		if !present[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return &ValidationError{
			MessageType: key,
			Missing:     missing,
		}
	}

	return nil
}
