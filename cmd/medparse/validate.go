package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/medparse/medparse"
)

func runValidate(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	msgType := fs.String("type", "", "Explicit message type to validate (e.g. ADT_A01, ORU_R01; default: read from MSH-9)")
	fields := fs.String("fields", "", "Comma-separated list of required field paths (e.g. 'PID-3,PID-5')")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse validate [options] [file]

Options:
  -type string    Expected message type (e.g. ADT_A01, ORU_R01; inferred from MSH-9 if omitted)
  -fields string  Comma-separated list of required field paths (e.g. 'PID-3,PID-5.1')

Arguments:
  [file]          Path to HL7 file (reads from stdin if omitted or '-')

Examples:
  cat msg.hl7 | medparse validate
  medparse validate --type ADT_A01 msg.hl7
  medparse validate --fields "PID-3,PID-5.1" msg.hl7
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	parsedArgs := fs.Args()
	var filename string
	if len(parsedArgs) >= 1 {
		filename = parsedArgs[0]
	}

	data, err := readInput(filename, stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	msg, err := medparse.Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse HL7 message: %w", err)
	}

	// Validate required segments
	targetType := *msgType
	if targetType != "" {
		// If explicit type provided, mutate MSH-9 components temporarily
		origMSH9_1, _ := msg.Get("MSH-9.1")
		origMSH9_2, _ := msg.Get("MSH-9.2")
		parts := strings.Split(targetType, "_")
		if len(parts) >= 2 {
			_ = msg.Set("MSH-9.1", parts[0])
			_ = msg.Set("MSH-9.2", parts[1])
		} else {
			_ = msg.Set("MSH-9.1", targetType)
			_ = msg.Set("MSH-9.2", "")
		}
		valErr := msg.Validate()
		_ = msg.Set("MSH-9.1", origMSH9_1)
		_ = msg.Set("MSH-9.2", origMSH9_2)
		if valErr != nil {
			return valErr
		}
	} else {
		if err := msg.Validate(); err != nil {
			return err
		}
		ev, tr, _ := msg.MessageType()
		if ev != "" {
			if tr != "" {
				targetType = ev + "_" + tr
			} else {
				targetType = ev
			}
		} else {
			targetType = "HL7"
		}
	}

	// Validate required fields if specified
	if *fields != "" {
		rawPaths := strings.Split(*fields, ",")
		var paths []string
		for _, p := range rawPaths {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			if err := msg.ValidateRequiredFields(paths...); err != nil {
				return fmt.Errorf("missing or empty required field: %w", err)
			}
		}
	}

	fmt.Fprintf(stdout, "OK: valid %s\n", targetType)
	return nil
}
