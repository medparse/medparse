package medparse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTimestamp parses an HL7v2 timestamp string into a time.Time.
//
// Handles full and partial timestamps:
//   - "20230101" → 2023-01-01 00:00:00
//   - "20230101120000" → 2023-01-01 12:00:00
//   - "20230101120000.123" → 2023-01-01 12:00:00.123
//   - "20230101120000-0500" → 2023-01-01 12:00:00 -0500
func ParseTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 4 {
		return time.Time{}, &ParseError{Msg: fmt.Sprintf("timestamp too short: '%s'", raw)}
	}

	dtStr, tzOffset := splitTimezone(raw)

	year, month, day, hour, minute, second, micros, err := parseTimestampParts(dtStr)
	if err != nil {
		return time.Time{}, err
	}

	nsec := micros * 1000 // microseconds → nanoseconds

	var loc *time.Location
	if tzOffset != nil {
		loc = time.FixedZone("", *tzOffset*60)
	} else {
		loc = time.UTC
	}

	t := time.Date(year, time.Month(month), day, hour, minute, second, nsec, loc)
	return t, nil
}

// ParseDate parses an HL7v2 timestamp into just a date (time component is zeroed).
func ParseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	dtStr, _ := splitTimezone(raw)

	if len(dtStr) < 8 {
		return time.Time{}, &ParseError{Msg: fmt.Sprintf("timestamp too short for date: '%s'", raw)}
	}

	year, err := strconv.Atoi(dtStr[:4])
	if err != nil {
		return time.Time{}, &ParseError{Msg: fmt.Sprintf("invalid year in '%s'", raw)}
	}
	month, err := strconv.Atoi(dtStr[4:6])
	if err != nil {
		return time.Time{}, &ParseError{Msg: fmt.Sprintf("invalid month in '%s'", raw)}
	}
	day, err := strconv.Atoi(dtStr[6:8])
	if err != nil {
		return time.Time{}, &ParseError{Msg: fmt.Sprintf("invalid day in '%s'", raw)}
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// splitTimezone splits a timezone suffix (+HHMM or -HHMM) from the timestamp.
// Returns (datetime_part, optional_offset_in_minutes).
func splitTimezone(raw string) (string, *int) {
	if len(raw) < 5 {
		return raw, nil
	}

	tzStart := len(raw) - 5
	tzPart := raw[tzStart:]

	if tzPart[0] != '+' && tzPart[0] != '-' {
		return raw, nil
	}

	hours, err1 := strconv.Atoi(tzPart[1:3])
	mins, err2 := strconv.Atoi(tzPart[3:5])
	if err1 != nil || err2 != nil {
		return raw, nil
	}

	sign := 1
	if tzPart[0] == '-' {
		sign = -1
	}
	offset := sign * (hours*60 + mins)

	return raw[:tzStart], &offset
}

// parseTimestampParts extracts year, month, day, hour, minute, second, microseconds
// from an HL7 timestamp string (without timezone).
func parseTimestampParts(s string) (year, month, day, hour, minute, second, micros int, err error) {
	n := len(s)

	if n < 4 {
		return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("timestamp too short: '%s'", s)}
	}

	year, _ = strconv.Atoi(s[:4])
	month = 1
	day = 1

	if n >= 6 {
		month, _ = strconv.Atoi(s[4:6])
	}
	if n >= 8 {
		day, _ = strconv.Atoi(s[6:8])
	}
	if n >= 10 {
		hour, _ = strconv.Atoi(s[8:10])
	}
	if n >= 12 {
		minute, _ = strconv.Atoi(s[10:12])
	}
	if n >= 14 {
		second, _ = strconv.Atoi(s[12:14])
	}

	// Fractional seconds (after the dot).
	if n > 15 && s[14] == '.' {
		fracStr := s[15:]
		// Pad or truncate to 6 digits (microseconds).
		if len(fracStr) > 6 {
			fracStr = fracStr[:6]
		}
		for len(fracStr) < 6 {
			fracStr += "0"
		}
		micros, _ = strconv.Atoi(fracStr)
	}

	return
}
