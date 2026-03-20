package mapping_test

import (
	"fmt"

	medparse "github.com/medparse/medparse"
	"github.com/medparse/medparse/mapping"
)

func ExampleFieldMap() {
	// Define a site-specific mapping.
	epic := mapping.FieldMap{
		"last_name":  "PID-5-1",
		"first_name": "PID-5-2",
		"mrn":        "PID-3-1",
		"gender":     "PID-8",
	}

	msg, _ := medparse.Parse(
		"MSH|^~\\&|EPIC|HOSPITAL|RECV|FAC|20230101||ADT^A01|1|P|2.5\r" +
			"PID|1||MRN123^^^MRN||DOE^JOHN^M||19800101|M",
	)

	name, _ := epic.Get(msg, "last_name")
	mrn, _ := epic.Get(msg, "mrn")
	fmt.Println(name, mrn)
	// Output: DOE MRN123
}

func ExampleExtractor() {
	// Build an extractor with both simple paths and complex logic.
	ext := mapping.NewExtractor(mapping.FieldMap{
		"last_name": "PID-5-1",
		"mrn":       "PID-3-1",
	}).WithFunc("primary_dx",
		// Find DG1 where diagnosis type (DG1-6) = "A" (admitting),
		// then return the diagnosis code (DG1-3.1).
		mapping.SegmentWhere("DG1", 6, "A", 3, 1),
	)

	msg, _ := medparse.Parse(
		"MSH|^~\\&|S|F|R|F|20230101||ADT^A01|1|P|2.5\r" +
			"PID|1||MRN123||DOE^JOHN\r" +
			"DG1|1||I10^Hypertension^ICD10||20230101|A\r" +
			"DG1|2||M54.5^Back Pain^ICD10||20230101|W",
	)

	name, _ := ext.Get(msg, "last_name")
	dx, _ := ext.Get(msg, "primary_dx")
	fmt.Println(name, dx)
	// Output: DOE I10
}

func ExampleFieldMap_Merge() {
	base := mapping.FieldMap{
		"last_name": "PID-5-1",
		"mrn":       "PID-3-1",
	}

	// Site-specific override.
	siteOverride := mapping.FieldMap{
		"mrn": "PID-18-1", // this site uses PID-18 for MRN
	}

	merged := base.Merge(siteOverride)
	fmt.Println(merged["mrn"])
	// Output: PID-18-1
}
