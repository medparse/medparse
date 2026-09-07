package main

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

const sampleHL7 = "MSH|^~\\&|SEND_APP|SEND_FAC|RECV_APP|RECV_FAC|20230101120000||ADT^A01|MSG001|P|2.5\rEVN||20230101120000\rPID|1||12345^^^HOSP^MR~98765^^^STATE^DL||SMITH^JOHN^A||19800101|M|||123 MAIN ST^^SPRINGFIELD^IL^62701\rPV1|1|I|WARD1^ROOM2^BED3"

func TestCLI_Get(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runGet([]string{"PID-5.1"}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("runGet failed: %v", err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "SMITH" {
			t.Errorf("got %q, want %q", got, "SMITH")
		}
	})

	t.Run("field repetition dot syntax", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runGet([]string{"PID-3(1)-1"}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("runGet failed: %v", err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "98765" {
			t.Errorf("got %q, want %q", got, "98765")
		}
	})

	t.Run("get -all repetitions", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runGet([]string{"-all", "PID-3-1"}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("runGet -all failed: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 2 || lines[0] != "12345" || lines[1] != "98765" {
			t.Errorf("got %v, want [12345 98765]", lines)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runGet([]string{"PID-99"}, stdin, &stdout, &stderr)
		if err == nil {
			t.Fatalf("expected error for non-existent path")
		}
	})
}

func TestCLI_Set(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(sampleHL7)

	err := runSet([]string{"PID-5.1", "WILLIAMS"}, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runSet failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "WILLIAMS^JOHN^A") {
		t.Errorf("expected updated message to contain WILLIAMS^JOHN^A, got:\n%s", out)
	}
}

func TestCLI_Validate(t *testing.T) {
	t.Run("valid ADT_A01 message", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runValidate([]string{}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("expected valid message, got err: %v", err)
		}
		if !strings.Contains(stdout.String(), "OK: valid ADT_A01") {
			t.Errorf("got output %q, expected 'OK: valid ADT_A01'", stdout.String())
		}
	})

	t.Run("explicit type with missing segment", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		// sampleHL7 lacks MRG segment required by ADT_A40
		stdin := strings.NewReader(sampleHL7)

		err := runValidate([]string{"--type", "ADT_A40"}, stdin, &stdout, &stderr)
		if err == nil {
			t.Fatalf("expected validation error for missing MRG segment in ADT_A40")
		}
	})

	t.Run("required fields pass", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runValidate([]string{"--fields", "PID-3.1,PID-5.1"}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("expected valid required fields, got: %v", err)
		}
	})

	t.Run("required fields missing", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runValidate([]string{"--fields", "PID-19"}, stdin, &stdout, &stderr)
		if err == nil {
			t.Fatalf("expected validation error for empty PID-19")
		}
	})
}

func TestCLI_ACK(t *testing.T) {
	t.Run("default AA", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runACK([]string{}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("runACK failed: %v", err)
		}

		out := stdout.String()
		if !strings.Contains(out, "MSA|AA|MSG001") {
			t.Errorf("expected MSA|AA|MSG001 in ACK, got:\n%s", out)
		}
		if !strings.Contains(out, "RECV_APP|RECV_FAC|SEND_APP|SEND_FAC") {
			t.Errorf("expected sender/receiver swapped in MSH, got:\n%s", out)
		}
	})

	t.Run("error code AE with text", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(sampleHL7)

		err := runACK([]string{"-code", "AE", "-text", "Patient not found"}, stdin, &stdout, &stderr)
		if err != nil {
			t.Fatalf("runACK failed: %v", err)
		}

		out := stdout.String()
		if !strings.Contains(out, "MSA|AE|MSG001|Patient not found") {
			t.Errorf("expected MSA|AE|MSG001|Patient not found in ACK, got:\n%s", out)
		}
	})
}

func TestCLI_MLLP_SendAndListen(t *testing.T) {
	// Find an open port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close() // Close so runMLLPListen can bind it

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var srvStdout, srvStderr bytes.Buffer
	srvErrCh := make(chan error, 1)

	parts := strings.Split(addr, ":")
	host := parts[0]
	portStr := parts[1]

	go func() {
		srvErrCh <- runMLLPListen(ctx, []string{"-addr", host, "-port", portStr, "-ack=true"}, &srvStdout, &srvStderr)
	}()

	// Wait for server to bind
	var clientErr error
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			clientErr = nil
			break
		}
		clientErr = err
	}
	if clientErr != nil {
		t.Fatalf("server failed to start listening: %v", clientErr)
	}

	// Send message using runMLLPSend
	var clientStdout, clientStderr bytes.Buffer
	stdin := strings.NewReader(sampleHL7)
	err = runMLLPSend([]string{"-addr", addr, "-timeout", "2s"}, stdin, &clientStdout, &clientStderr)
	if err != nil {
		t.Fatalf("runMLLPSend failed: %v", err)
	}

	// Verify client got ACK back
	clientResp := clientStdout.String()
	if !strings.Contains(clientResp, "MSA|AA|MSG001") {
		t.Errorf("expected client to receive ACK with MSA|AA|MSG001, got:\n%s", clientResp)
	}

	// Verify server logged the incoming message
	if !strings.Contains(srvStdout.String(), "SMITH^JOHN^A") {
		t.Errorf("expected server stdout to contain received HL7, got:\n%s", srvStdout.String())
	}

	// Cancel context to shut down server
	cancel()
	select {
	case err := <-srvErrCh:
		if err != nil {
			t.Fatalf("server returned error on shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server failed to shut down in time")
	}
}
