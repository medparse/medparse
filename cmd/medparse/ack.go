package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/medparse/medparse"
)

func runACK(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("ack", flag.ContinueOnError)
	fs.SetOutput(stderr)

	code := fs.String("code", "AA", "Acknowledgment code: AA (Accept), AE (Error), AR (Reject)")
	text := fs.String("text", "", "Optional acknowledgment text message for MSA-3")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse ack [options] [file]

Options:
  -code string  Acknowledgment code: AA, AE, AR (default "AA")
  -text string  Optional text message for MSA-3

Arguments:
  [file]        Path to HL7 file (reads from stdin if omitted or '-')

Examples:
  cat msg.hl7 | medparse ack
  cat msg.hl7 | medparse ack -code AE -text "Invalid MRN format"
  medparse ack patient.hl7
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	upperCode := strings.ToUpper(strings.TrimSpace(*code))
	switch upperCode {
	case "AA", "AE", "AR", "CA", "CE", "CR":
	default:
		return fmt.Errorf("invalid acknowledgment code %q: must be AA, AE, or AR", *code)
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

	ackMsg, err := msg.BuildACK(upperCode, *text)
	if err != nil {
		return fmt.Errorf("failed to build ACK: %w", err)
	}

	fmt.Fprintln(stdout, ackMsg.String())
	return nil
}
