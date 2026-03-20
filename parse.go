package medparse

import "strings"

// Parse parses a raw HL7v2 message string into a Message.
//
// Handles MLLP-framed input automatically. Supports \r, \n, and \r\n
// segment delimiters.
func Parse(raw string) (*Message, error) {
	// Strip MLLP framing if present.
	raw = StripMLLP(raw)

	if len(raw) == 0 {
		return nil, &ParseError{Msg: "empty message"}
	}

	// Split into segment strings.
	segStrs := splitSegments(raw)
	if len(segStrs) == 0 {
		return nil, &ParseError{Msg: "no segments found"}
	}

	// Extract encoding characters from MSH.
	first := segStrs[0]
	if !strings.HasPrefix(first, "MSH") {
		preview := first
		if len(preview) > 10 {
			preview = preview[:10]
		}
		return nil, &ParseError{Msg: "message must start with MSH segment, got: '" + preview + "'"}
	}

	enc, err := extractEncodingChars(first)
	if err != nil {
		return nil, err
	}

	// Parse all segments.
	segments := make([]Segment, 0, len(segStrs))
	for _, segStr := range segStrs {
		if len(segStr) == 0 {
			continue
		}
		seg, err := parseSegment(segStr, enc)
		if err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}

	return &Message{
		Raw:      raw,
		Segments: segments,
		Enc:      *enc,
	}, nil
}

// extractEncodingChars extracts encoding characters from the MSH segment.
//
// MSH layout: MSH|^~\&|...
//
//	Position 3:   field separator (|)
//	Position 4:   component separator (^)
//	Position 5:   repetition separator (~)
//	Position 6:   escape character (\)
//	Position 7:   sub-component separator (&)
func extractEncodingChars(msh string) (*EncodingChars, error) {
	if len(msh) < 8 {
		return nil, &ParseError{Msg: "MSH segment too short to extract encoding characters"}
	}

	return &EncodingChars{
		FieldSep:        msh[3],
		ComponentSep:    msh[4],
		RepetitionSep:   msh[5],
		EscapeChar:      msh[6],
		SubComponentSep: msh[7],
	}, nil
}

// splitSegments splits a raw message into segment strings, handling \r, \n, and \r\n.
func splitSegments(raw string) []string {
	var result []string
	for _, s := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\r' || r == '\n'
	}) {
		if len(s) > 0 {
			result = append(result, s)
		}
	}
	return result
}

// parseSegment parses a single segment string into a Segment.
func parseSegment(segStr string, enc *EncodingChars) (Segment, error) {
	isMSH := strings.HasPrefix(segStr, "MSH")
	sep := enc.FieldSep

	// Split into name and fields.
	idx := strings.IndexByte(segStr, sep)
	if idx < 0 {
		// Segment with just a name, no fields.
		return Segment{Name: segStr, Fields: nil}, nil
	}

	name := segStr[:idx]
	fieldsStr := segStr[idx+1:]

	var fields []Field

	if isMSH {
		// MSH-1 = field separator itself.
		sepField := Field{
			Value: string(sep),
			Components: []Component{{
				Value:         string(sep),
				SubComponents: []string{string(sep)},
			}},
		}

		rawFields := strings.Split(fieldsStr, string(sep))
		fields = make([]Field, 0, len(rawFields)+1)
		fields = append(fields, sepField)

		for _, rf := range rawFields {
			fields = append(fields, parseField(rf, enc))
		}
	} else {
		rawFields := strings.Split(fieldsStr, string(sep))
		fields = make([]Field, 0, len(rawFields))
		for _, rf := range rawFields {
			fields = append(fields, parseField(rf, enc))
		}
	}

	return Segment{Name: name, Fields: fields}, nil
}

// parseField parses a single field string, handling repetitions and components.
func parseField(raw string, enc *EncodingChars) Field {
	// Check for repetitions.
	repParts := strings.Split(raw, string(enc.RepetitionSep))

	var repetitions []Field
	if len(repParts) > 1 {
		repetitions = make([]Field, len(repParts))
		for i, r := range repParts {
			repetitions[i] = parseSingleField(r, enc)
		}
	}

	// Parse the first (or only) value.
	field := parseSingleField(repParts[0], enc)
	field.Repetitions = repetitions

	// The top-level value is the full raw string (including repetition separators).
	field.Value = decodeEscapes(raw, enc)

	return field
}

// parseSingleField parses a single field value (no repetition handling) into components.
func parseSingleField(raw string, enc *EncodingChars) Field {
	compParts := strings.Split(raw, string(enc.ComponentSep))

	components := make([]Component, len(compParts))
	for i, c := range compParts {
		components[i] = parseComponent(c, enc)
	}

	return Field{
		Value:      decodeEscapes(raw, enc),
		Components: components,
	}
}

// parseComponent parses a component string, extracting sub-components.
func parseComponent(raw string, enc *EncodingChars) Component {
	subParts := strings.Split(raw, string(enc.SubComponentSep))
	subs := make([]string, len(subParts))
	for i, s := range subParts {
		subs[i] = decodeEscapes(s, enc)
	}

	return Component{
		Value:         decodeEscapes(raw, enc),
		SubComponents: subs,
	}
}
