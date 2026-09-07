package medparse

import (
	"strings"
	"sync"
)

var (
	rulesMu          sync.RWMutex
	requiredSegments = map[string][]string{
		// ADT messages
		"ADT_A01": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A02": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A03": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A04": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A05": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A06": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A07": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A08": {"MSH", "EVN", "PID", "PV1"},
		"ADT_A31": {"MSH", "EVN", "PID"},
		"ADT_A40": {"MSH", "EVN", "PID", "MRG"},

		// Order messages
		"ORM_O01": {"MSH", "PID", "ORC"},
		"OML_O21": {"MSH", "PID", "ORC", "OBR"},

		// Result messages
		"ORU_R01": {"MSH", "PID", "OBR", "OBX"},

		// Scheduling
		"SIU_S12": {"MSH", "SCH", "PID"},

		// Master file
		"MFN_M01": {"MSH", "MFI"},

		// Document Management
		"MDM_T02": {"MSH", "EVN", "PID", "PV1", "TXA"},

		// Financial / Billing
		"DFT_P03": {"MSH", "EVN", "PID", "FT1"},
		"BAR_P01": {"MSH", "EVN", "PID"},

		// Vaccines
		"VXU_V04": {"MSH", "PID", "RXA"},

		// Pharmacy
		"RAS_O17": {"MSH", "PID", "ORC", "RXA"},
		"RDE_O11": {"MSH", "PID", "ORC", "RXO"},

		// Acknowledgment
		"ACK": {"MSH", "MSA"},
	}
)

// RegisterRequiredSegments registers or overrides required segments for a message type.
// messageType can be specific (e.g. "ADT_A01") or generic (e.g. "ACK").
// This function is safe for concurrent use.
func RegisterRequiredSegments(messageType string, segments []string) {
	rulesMu.Lock()
	defer rulesMu.Unlock()
	segsCopy := make([]string, len(segments))
	copy(segsCopy, segments)
	requiredSegments[messageType] = segsCopy
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

	key := eventType + "_" + trigger

	rulesMu.RLock()
	required, ok := requiredSegments[key]
	if !ok {
		required, ok = requiredSegments[eventType]
		if !ok {
			rulesMu.RUnlock()
			return nil // no rules for this message type
		}
		key = eventType
	}
	segs := make([]string, len(required))
	copy(segs, required)
	rulesMu.RUnlock()

	// Build set of segment names present in the message.
	present := make(map[string]bool, len(m.Segments))
	for _, seg := range m.Segments {
		present[seg.Name] = true
	}

	// Check for missing required segments.
	var missing []string
	for _, name := range segs {
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

// ValidateRequiredFields checks that specific paths in the message exist and contain non-empty values.
// Returns a ValidationError for the first missing or empty field path found.
func (m *Message) ValidateRequiredFields(paths ...string) error {
	for _, path := range paths {
		val, err := m.Get(path)
		if err != nil || strings.TrimSpace(val) == "" {
			return &ValidationError{
				MessageType: path,
				Missing:     []string{path},
			}
		}
	}
	return nil
}
