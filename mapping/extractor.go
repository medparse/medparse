package mapping

import (
	"fmt"

	medparse "github.com/medparse/medparse"
)

// ExtractFunc is a custom extraction function that retrieves a value from a message.
// Use this for complex logic that can't be expressed as a single terser path.
type ExtractFunc func(msg *medparse.Message) (string, error)

// Extractor combines a FieldMap for simple path lookups with custom
// ExtractFunc functions for complex extractions.
//
// When Get is called, it first checks for a custom function, then falls
// back to the FieldMap.
type Extractor struct {
	// Fields provides simple path-based mappings.
	Fields FieldMap

	// Funcs provides custom extraction functions for keys that require
	// more complex logic than a single terser path.
	Funcs map[string]ExtractFunc
}

// NewExtractor creates an Extractor with the given FieldMap and no custom functions.
func NewExtractor(fields FieldMap) *Extractor {
	return &Extractor{
		Fields: fields,
		Funcs:  make(map[string]ExtractFunc),
	}
}

// WithFunc registers a custom extraction function for a key.
// Returns the Extractor for chaining.
func (e *Extractor) WithFunc(key string, fn ExtractFunc) *Extractor {
	e.Funcs[key] = fn
	return e
}

// Get retrieves a value by key. Custom functions take priority over FieldMap paths.
func (e *Extractor) Get(msg *medparse.Message, key string) (string, error) {
	// Check custom functions first.
	if fn, ok := e.Funcs[key]; ok {
		return fn(msg)
	}

	// Fall back to FieldMap.
	return e.Fields.Get(msg, key)
}

// GetAll retrieves all values (both FieldMap and custom functions).
// Fields that error are set to empty string.
func (e *Extractor) GetAll(msg *medparse.Message) map[string]string {
	result := e.Fields.GetAll(msg)

	// Overlay custom function results.
	for key, fn := range e.Funcs {
		val, err := fn(msg)
		if err != nil {
			val = ""
		}
		result[key] = val
	}

	return result
}

// ---------------------------------------------------------------------------
// Common extraction helpers
// ---------------------------------------------------------------------------

// FirstSegmentField returns an ExtractFunc that gets a field from the first
// segment matching the given name.
func FirstSegmentField(segName string, fieldIdx, compIdx int) ExtractFunc {
	return func(msg *medparse.Message) (string, error) {
		seg, err := msg.Segment(segName)
		if err != nil {
			return "", err
		}
		f, err := seg.Field(fieldIdx)
		if err != nil {
			return "", err
		}
		if compIdx > 0 {
			comp, err := f.Component(compIdx)
			if err != nil {
				return "", err
			}
			return comp.Value, nil
		}
		return f.Value, nil
	}
}

// SegmentWhere returns an ExtractFunc that finds a segment where a given field
// matches a value, then extracts from a target field.
//
// Example: find DG1 where DG1-6 = "A", then return DG1-3.1:
//
//	SegmentWhere("DG1", 6, "A", 3, 1)
func SegmentWhere(segName string, whereField int, whereValue string, targetField, targetComp int) ExtractFunc {
	return func(msg *medparse.Message) (string, error) {
		var result string
		var found bool

		msg.EachSegment(segName, func(i int, seg *medparse.Segment) error {
			f, err := seg.Field(whereField)
			if err != nil {
				return nil // skip
			}
			if f.Value == whereValue {
				tf, err := seg.Field(targetField)
				if err != nil {
					return nil
				}
				if targetComp > 0 {
					comp, err := tf.Component(targetComp)
					if err != nil {
						return nil
					}
					result = comp.Value
				} else {
					result = tf.Value
				}
				found = true
				return errStop
			}
			return nil
		})

		if !found {
			return "", &errNotMatched{segName: segName, field: whereField, value: whereValue}
		}
		return result, nil
	}
}

// internal sentinel for stopping EachSegment iteration.
var errStop = &stopError{}

type stopError struct{}

func (e *stopError) Error() string { return "stop" }

type errNotMatched struct {
	segName string
	field   int
	value   string
}

func (e *errNotMatched) Error() string {
	return fmt.Sprintf("no %s segment found where field %d = '%s'", e.segName, e.field, e.value)
}
