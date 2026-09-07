package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/medparse/medparse"
)

func runGet(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(stderr)

	all := fs.Bool("all", false, "Retrieve all occurrences across segment and field repetitions")
	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse get [options] <path> [file]

Options:
  -all    Retrieve all occurrences across repetitions (prints one per line)

Arguments:
  <path>  HL7 path (e.g. PID-5.1, PID-3(0)-1, OBX-5)
  [file]  Path to HL7 file (reads from stdin if omitted or '-')

Examples:
  cat msg.hl7 | medparse get PID-5.1
  medparse get -all OBX-5 lab_results.hl7
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) < 1 {
		fs.Usage()
		return fmt.Errorf("missing required argument: <path>")
	}

	path := parsedArgs[0]
	var filename string
	if len(parsedArgs) >= 2 {
		filename = parsedArgs[1]
	}

	data, err := readInput(filename, stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	msg, err := medparse.Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse HL7 message: %w", err)
	}

	if *all {
		vals, err := msg.GetAll(path)
		if err != nil {
			return err
		}
		if len(vals) == 0 {
			return fmt.Errorf("no values found for path: %s", path)
		}
		for _, v := range vals {
			fmt.Fprintln(stdout, v)
		}
		return nil
	}

	val, err := msg.Get(path)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, val)
	return nil
}
