package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/medparse/medparse"
)

func runSet(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse set <path> <value> [file]

Arguments:
  <path>   HL7 path (e.g. PID-5.1, PID-3(0)-1)
  <value>  Value to set
  [file]   Path to HL7 file (reads from stdin if omitted or '-')

Examples:
  cat msg.hl7 | medparse set PID-5.1 "SMITH"
  medparse set PID-3-1 "MRN99999" patient.hl7
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) < 2 {
		fs.Usage()
		return fmt.Errorf("missing required arguments: <path> <value>")
	}

	path := parsedArgs[0]
	val := parsedArgs[1]
	var filename string
	if len(parsedArgs) >= 3 {
		filename = parsedArgs[2]
	}

	data, err := readInput(filename, stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	msg, err := medparse.Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse HL7 message: %w", err)
	}

	if err := msg.Set(path, val); err != nil {
		return fmt.Errorf("failed to set path %q: %w", path, err)
	}

	fmt.Fprintln(stdout, msg.String())
	return nil
}
