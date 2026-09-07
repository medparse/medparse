package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const version = "0.4.0"

func printUsage() {
	fmt.Fprintf(os.Stderr, `medparse - High-performance HL7v2 CLI toolkit

Usage:
  medparse <command> [arguments]

Commands:
  get       Extract field values using Terser path syntax (e.g. PID-5.1 or PID-3(0)-1)
  set       Mutate a field value and output the updated HL7 message
  validate  Validate required segments and fields in an HL7 message
  ack       Generate an acknowledgment (ACK) for an HL7 message
  mllp      Send or listen for MLLP-framed HL7 messages over TCP
  version   Print medparse version
  help      Show help for a command

Run 'medparse <command> -h' for more information on a command.
`)
}

func readInput(filename string, stdin io.Reader) ([]byte, error) {
	if filename == "" || filename == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(filename)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "get":
		err = runGet(args, os.Stdin, os.Stdout, os.Stderr)
	case "set":
		err = runSet(args, os.Stdin, os.Stdout, os.Stderr)
	case "validate":
		err = runValidate(args, os.Stdin, os.Stdout, os.Stderr)
	case "ack":
		err = runACK(args, os.Stdin, os.Stdout, os.Stderr)
	case "mllp":
		err = runMLLP(args, os.Stdin, os.Stdout, os.Stderr)
	case "version", "-v", "--version":
		fmt.Printf("medparse v%s\n", version)
		return
	case "help", "-h", "--help":
		if len(args) > 0 {
			switch args[0] {
			case "get":
				runGet([]string{"-h"}, os.Stdin, os.Stdout, os.Stderr)
			case "set":
				runSet([]string{"-h"}, os.Stdin, os.Stdout, os.Stderr)
			case "validate":
				runValidate([]string{"-h"}, os.Stdin, os.Stdout, os.Stderr)
			case "ack":
				runACK([]string{"-h"}, os.Stdin, os.Stdout, os.Stderr)
			case "mllp":
				runMLLP([]string{"-h"}, os.Stdin, os.Stdout, os.Stderr)
			default:
				printUsage()
			}
			return
		}
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
