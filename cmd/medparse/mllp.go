package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/medparse/medparse"
)

func runMLLP(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		printMLLPUsage(stderr)
		return fmt.Errorf("missing mllp subcommand: 'send' or 'listen'")
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "send":
		return runMLLPSend(subargs, stdin, stdout, stderr)
	case "listen":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runMLLPListen(ctx, subargs, stdout, stderr)
	case "-h", "--help", "help":
		printMLLPUsage(stderr)
		return nil
	default:
		printMLLPUsage(stderr)
		return fmt.Errorf("unknown mllp subcommand: %s", subcmd)
	}
}

func printMLLPUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  medparse mllp send --addr <host:port> [options] [file]
  medparse mllp listen [options]

Subcommands:
  send    Send an HL7 message framed with MLLP over TCP and await the ACK response
  listen  Listen on a TCP port for incoming MLLP-framed HL7 messages

Run 'medparse mllp <subcommand> -h' for more information.
`)
}

func runMLLPSend(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mllp send", flag.ContinueOnError)
	fs.SetOutput(stderr)

	addr := fs.String("addr", "", "Remote TCP address (e.g. 'localhost:2575' or '10.0.1.5:2575')")
	timeout := fs.Duration("timeout", 5*time.Second, "Network dial and read timeout")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse mllp send --addr <host:port> [options] [file]

Options:
  -addr string        Target MLLP server address (required)
  -timeout duration   Timeout for connect and read (default: 5s)

Arguments:
  [file]              Path to HL7 message file (reads from stdin if omitted or '-')

Examples:
  cat msg.hl7 | medparse mllp send --addr localhost:2575
  medparse mllp send --addr hl7server.local:2575 adt_a01.hl7
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *addr == "" {
		fs.Usage()
		return fmt.Errorf("missing required option: -addr")
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

	if len(data) == 0 {
		return fmt.Errorf("input message is empty")
	}

	conn, err := net.DialTimeout("tcp", *addr, *timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", *addr, err)
	}
	defer conn.Close()

	if *timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(*timeout))
	}

	writer := medparse.NewMLLPWriter(conn)
	if err := writer.WriteMessage(data); err != nil {
		return fmt.Errorf("failed to send MLLP message: %w", err)
	}

	reader := medparse.NewMLLPReader(conn)
	resp, err := reader.ReadString()
	if err != nil {
		return fmt.Errorf("failed to read MLLP response: %w", err)
	}

	fmt.Fprintln(stdout, resp)
	return nil
}

func runMLLPListen(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mllp listen", flag.ContinueOnError)
	fs.SetOutput(stderr)

	addr := fs.String("addr", "0.0.0.0", "Bind IP address")
	port := fs.Int("port", 2575, "TCP port to listen on")
	ack := fs.Bool("ack", true, "Automatically send standard ACK response to incoming messages")
	ackCode := fs.String("code", "AA", "Acknowledgment code for ACK responses (AA, AE, AR)")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage:
  medparse mllp listen [options]

Options:
  -addr string   Bind IP address (default "0.0.0.0")
  -port int      TCP port to listen on (default 2575)
  -ack           Automatically reply with standard ACK (default true)
  -code string   ACK code to return: AA, AE, AR (default "AA")

Examples:
  medparse mllp listen --port 2575
  medparse mllp listen --port 2575 --ack=false
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	bindAddr := fmt.Sprintf("%s:%d", *addr, *port)
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", bindAddr, err)
	}
	defer listener.Close()

	fmt.Fprintf(stderr, "medparse MLLP server listening on %s (press Ctrl+C to stop)\n", bindAddr)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				fmt.Fprintf(stderr, "accept error: %v\n", err)
				continue
			}
		}

		go handleMLLPConn(conn, *ack, *ackCode, stdout, stderr)
	}
}

func handleMLLPConn(conn net.Conn, sendACK bool, ackCode string, stdout, stderr io.Writer) {
	defer conn.Close()

	reader := medparse.NewMLLPReader(conn)
	writer := medparse.NewMLLPWriter(conn)

	for {
		msgBytes, err := reader.ReadMessage()
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(stderr, "error reading from %s: %v\n", conn.RemoteAddr(), err)
			}
			return
		}

		// Output received message
		fmt.Fprintf(stdout, "%s\n", string(msgBytes))

		if sendACK {
			msg, parseErr := medparse.Parse(string(msgBytes))
			var ackStr string
			if parseErr == nil {
				ackMsg, ackErr := msg.BuildACK(ackCode, "Message received")
				if ackErr == nil {
					ackStr = ackMsg.String()
				}
			}

			if ackStr == "" {
				// Fallback simple ACK
				now := time.Now().Format("20060102150405")
				ackStr = fmt.Sprintf("MSH|^~\\&|MEDPARSE|SYSTEM|||%s||ACK|ACK%s|P|2.5\rMSA|%s|UNKNOWN|Received", now, now, ackCode)
			}

			if writeErr := writer.WriteString(ackStr); writeErr != nil {
				fmt.Fprintf(stderr, "error sending ACK to %s: %v\n", conn.RemoteAddr(), writeErr)
				return
			}
		}
	}
}
