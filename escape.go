package medparse

import "strconv"

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
func decodeEscapes(value string, enc *EncodingChars) string {
	esc := enc.EscapeChar

	// Fast path: no escape character present.
	found := false
	for i := 0; i < len(value); i++ {
		if value[i] == esc {
			found = true
			break
		}
	}
	if !found {
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
						// Unknown escape — preserve as-is.
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
