// Package mapping provides a declarative field mapping layer for medparse.
//
// It lets you define site-specific or EHR-specific field mappings so the same
// extraction code works across different HL7 implementations (Epic, Cerner,
// Meditech, etc.) without hardcoding terser paths.
//
// For simple path-based mappings, use FieldMap. For complex extraction logic
// (e.g. "find the DG1 where type=A"), use Extractor.
package mapping

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	medparse "github.com/medparse/medparse"
)

// FieldMap maps logical field names to terser paths.
//
// Keys are your application's field names (e.g. "patient_last_name"),
// values are terser paths (e.g. "PID-5-1").
//
// Fallback paths can be specified using pipe (|) characters, e.g.:
//
//	"mrn": "PID-3(0)-1|PID-3-1"
//
// If the first path fails or returns an empty string, subsequent fallback paths are attempted.
type FieldMap map[string]string

// Get retrieves a value from the message using the mapped path for the given key.
// If the mapped path contains pipe-separated fallbacks, each is tried in order until
// a non-empty value is found.
func (fm FieldMap) Get(msg *medparse.Message, key string) (string, error) {
	path, ok := fm[key]
	if !ok {
		return "", fmt.Errorf("mapping: key '%s' not found in field map", key)
	}

	if strings.Contains(path, "|") {
		fallbacks := strings.Split(path, "|")
		var lastErr error
		for _, p := range fallbacks {
			val, err := msg.Get(p)
			if err == nil && val != "" {
				return val, nil
			}
			if err != nil {
				lastErr = err
			}
		}
		if lastErr != nil {
			return "", lastErr
		}
		return "", nil
	}

	return msg.Get(path)
}

// GetOrDefault retrieves a value from the message for the given key, returning defaultValue
// if the key is not defined, not found in the message, or empty.
func (fm FieldMap) GetOrDefault(msg *medparse.Message, key, defaultValue string) string {
	val, err := fm.Get(msg, key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// GetAll retrieves all mapped values from a message, returning a map of
// key → value. Fields that error (not found, out of range) are set to empty string.
func (fm FieldMap) GetAll(msg *medparse.Message) map[string]string {
	result := make(map[string]string, len(fm))
	for key, path := range fm {
		val, err := msg.Get(path)
		if err != nil {
			val = ""
		}
		result[key] = val
	}
	return result
}

// Set modifies a value in the message using the mapped path for the given key.
func (fm FieldMap) Set(msg *medparse.Message, key, value string) error {
	path, ok := fm[key]
	if !ok {
		return fmt.Errorf("mapping: key '%s' not found in field map", key)
	}
	return msg.Set(path, value)
}

// Merge returns a new FieldMap that combines the receiver with another.
// Keys in other override keys in the receiver.
func (fm FieldMap) Merge(other FieldMap) FieldMap {
	merged := make(FieldMap, len(fm)+len(other))
	for k, v := range fm {
		merged[k] = v
	}
	for k, v := range other {
		merged[k] = v
	}
	return merged
}

// Keys returns all defined field names in the map.
func (fm FieldMap) Keys() []string {
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	return keys
}

// LoadFieldMap loads a FieldMap from a JSON file.
//
// Expected format:
//
//	{
//	  "patient_last_name": "PID-5-1",
//	  "patient_first_name": "PID-5-2",
//	  "mrn": "PID-3-1"
//	}
func LoadFieldMap(path string) (FieldMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mapping: reading file: %w", err)
	}
	return ParseFieldMapJSON(data)
}

// ParseFieldMapJSON parses a FieldMap from JSON bytes.
func ParseFieldMapJSON(data []byte) (FieldMap, error) {
	var fm FieldMap
	if err := json.Unmarshal(data, &fm); err != nil {
		return nil, fmt.Errorf("mapping: parsing JSON: %w", err)
	}
	return fm, nil
}
