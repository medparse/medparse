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

// TimestampPrecision defines the output precision for HL7v2 timestamps.
type TimestampPrecision int

const (
	// PrecisionDay formats as YYYYMMDD.
	PrecisionDay TimestampPrecision = iota
	// PrecisionMinute formats as YYYYMMDDHHMM.
	PrecisionMinute
	// PrecisionSecond formats as YYYYMMDDHHMMSS (standard default).
	PrecisionSecond
	// PrecisionMilli formats as YYYYMMDDHHMMSS.sss.
	PrecisionMilli
	// PrecisionMicro formats as YYYYMMDDHHMMSS.ssssss.
	PrecisionMicro
)

// FormatTimestamp formats a time.Time into an HL7v2 timestamp string.
// If precision is omitted, PrecisionSecond (YYYYMMDDHHMMSS) is used.
func FormatTimestamp(t time.Time, prec ...TimestampPrecision) string {
	p := PrecisionSecond
	if len(prec) > 0 {
		p = prec[0]
	}
	switch p {
	case PrecisionDay:
		return t.Format("20060102")
	case PrecisionMinute:
		return t.Format("200601021504")
	case PrecisionMilli:
		return t.Format("20060102150405.000")
	case PrecisionMicro:
		return t.Format("20060102150405.000000")
	default:
		return t.Format("20060102150405")
	}
}

// FormatTimestampWithTZ formats a time.Time into an HL7v2 timestamp string with timezone offset (±HHMM).
func FormatTimestampWithTZ(t time.Time, prec ...TimestampPrecision) string {
	return FormatTimestamp(t, prec...) + t.Format("-0700")
}

// FormatDate formats a time.Time as an HL7v2 date (YYYYMMDD).
func FormatDate(t time.Time) string {
	return t.Format("20060102")
}

// parseTimestampParts extracts year, month, day, hour, minute, second, microseconds
// from an HL7 timestamp string (without timezone).
func parseTimestampParts(s string) (year, month, day, hour, minute, second, micros int, err error) {
	n := len(s)

	if n < 4 {
		return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("timestamp too short: '%s'", s)}
	}

	var errConv error
	year, errConv = strconv.Atoi(s[:4])
	if errConv != nil {
		return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid year in timestamp: '%s'", s)}
	}
	month = 1
	day = 1

	if n >= 6 {
		month, errConv = strconv.Atoi(s[4:6])
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid month in timestamp: '%s'", s)}
		}
	}
	if n >= 8 {
		day, errConv = strconv.Atoi(s[6:8])
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid day in timestamp: '%s'", s)}
		}
	}
	if n >= 10 {
		hour, errConv = strconv.Atoi(s[8:10])
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid hour in timestamp: '%s'", s)}
		}
	}
	if n >= 12 {
		minute, errConv = strconv.Atoi(s[10:12])
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid minute in timestamp: '%s'", s)}
		}
	}
	if n >= 14 {
		second, errConv = strconv.Atoi(s[12:14])
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid second in timestamp: '%s'", s)}
		}
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
		micros, errConv = strconv.Atoi(fracStr)
		if errConv != nil {
			return 0, 0, 0, 0, 0, 0, 0, &ParseError{Msg: fmt.Sprintf("invalid fractional seconds in timestamp: '%s'", s)}
		}
	}

	return
}
