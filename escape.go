package medparse

import (
	"strconv"
	"strings"
)

// Escape encodes special characters in a string into standard HL7v2 escape sequences.
//
// Special characters escaped:
//   - Field separator (|) → \F\
//   - Component separator (^) → \S\
//   - Sub-component separator (&) → \T\
//   - Repetition separator (~) → \R\
//   - Escape character (\) → \E\
//   - Newline (\n) / carriage return (\r) → \.br\
//
// If enc is nil, DefaultEncodingChars() is used.
func Escape(value string, enc *EncodingChars) string {
	if enc == nil {
		def := DefaultEncodingChars()
		enc = &def
	}

	esc := enc.EscapeChar
	fSep := enc.FieldSep
	cSep := enc.ComponentSep
	sSep := enc.SubComponentSep
	rSep := enc.RepetitionSep

	// Fast path: check if any character needs escaping.
	needsEscape := false
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b == esc || b == fSep || b == cSep || b == sSep || b == rSep || b == '\r' || b == '\n' {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return value
	}

	var b strings.Builder
	b.Grow(len(value) + 16)

	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch ch {
		case esc:
			b.WriteByte(esc)
			b.WriteByte('E')
			b.WriteByte(esc)
		case fSep:
			b.WriteByte(esc)
			b.WriteByte('F')
			b.WriteByte(esc)
		case cSep:
			b.WriteByte(esc)
			b.WriteByte('S')
			b.WriteByte(esc)
		case sSep:
			b.WriteByte(esc)
			b.WriteByte('T')
			b.WriteByte(esc)
		case rSep:
			b.WriteByte(esc)
			b.WriteByte('R')
			b.WriteByte(esc)
		case '\r':
			if i+1 < len(value) && value[i+1] == '\n' {
				i++ // skip \n in \r\n
			}
			b.WriteByte(esc)
			b.WriteString(".br")
			b.WriteByte(esc)
		case '\n':
			b.WriteByte(esc)
			b.WriteString(".br")
			b.WriteByte(esc)
		default:
			b.WriteByte(ch)
		}
	}

	return b.String()
}

// Unescape decodes HL7v2 escape sequences in a string.
// If enc is nil, DefaultEncodingChars() is used.
func Unescape(value string, enc *EncodingChars) string {
	if enc == nil {
		def := DefaultEncodingChars()
		enc = &def
	}
	return decodeEscapes(value, enc)
}

// decodeEscapes decodes HL7v2 escape sequences in a field value.
//
// Standard HL7 escape sequences:
//   - \F\ → field separator (|)
//   - \S\ → component separator (^)
//   - \T\ → sub-component separator (&)
//   - \R\ → repetition separator (~)
//   - \E\ → escape character (\)
//   - \X..\ → hex data
//   - \.br\ → line break (→ \n)
//   - \H\ / \N\ → highlight mode (preserved as is or stripped)
func decodeEscapes(value string, enc *EncodingChars) string {
	esc := enc.EscapeChar

	// Fast path: no escape character present.
	if strings.IndexByte(value, esc) < 0 {
		return value
	}

	result := make([]byte, 0, len(value))
	i := 0

	for i < len(value) {
		if value[i] == esc && i+2 < len(value) {
			// Look for the closing escape character.
			closeIdx := -1
			for j := i + 1; j < len(value); j++ {
				if value[j] == esc {
					closeIdx = j
					break
				}
			}

			if closeIdx > i+1 {
				seq := value[i+1 : closeIdx]
				switch seq {
				case "F":
					result = append(result, enc.FieldSep)
				case "S":
					result = append(result, enc.ComponentSep)
				case "T":
					result = append(result, enc.SubComponentSep)
				case "R":
					result = append(result, enc.RepetitionSep)
				case "E":
					result = append(result, enc.EscapeChar)
				case ".br":
					result = append(result, '\n')
				default:
					if len(seq) > 1 && seq[0] == 'X' {
						// Hex escape — decode hex bytes.
						hexStr := seq[1:]
						for j := 0; j+1 < len(hexStr); j += 2 {
							b, err := strconv.ParseUint(hexStr[j:j+2], 16, 8)
							if err == nil {
								result = append(result, byte(b))
							}
						}
					} else {
						// Unknown or highlight escape — preserve as-is.
						result = append(result, esc)
						result = append(result, seq...)
						result = append(result, esc)
					}
				}
				i = closeIdx + 1
			} else if closeIdx == i+1 {
				// Empty escape sequence — preserve as-is.
				result = append(result, esc, esc)
				i = closeIdx + 1
			} else {
				// No closing escape char found — preserve literally.
				result = append(result, value[i])
				i++
			}
		} else {
			result = append(result, value[i])
			i++
		}
	}

	return string(result)
}
