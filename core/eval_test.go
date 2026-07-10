package core

import (
	"io"
	"net"
	"testing"
)

// readResponse reads whatever a command handler wrote to the server side
// of a net.Pipe, from the client side.
func readResponse(t *testing.T, conn net.Conn) []byte {
	t.Helper()
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("read error: %v", err)
	}
	return buf[:n]
}

func TestEvalEXPIRE(t *testing.T) {
	setKey("expireme", "value")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalEXPIRE([]string{"expireme", "100"}, server); err != nil {
			t.Errorf("evalEXPIRE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":1\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}

	if ttl := ttlSeconds("expireme"); ttl <= 0 {
		t.Fatalf("expected positive ttl after EXPIRE, got %d", ttl)
	}
}

func TestEvalEXPIREMissingKey(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalEXPIRE([]string{"does-not-exist", "100"}, server); err != nil {
			t.Errorf("evalEXPIRE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":0\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}
}

func TestEvalEXPIRENonPositiveDeletesKey(t *testing.T) {
	setKey("expirenow", "value")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalEXPIRE([]string{"expirenow", "0"}, server); err != nil {
			t.Errorf("evalEXPIRE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":1\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}

	if _, ok := getKey("expirenow"); ok {
		t.Fatalf("expected key to be deleted immediately for non-positive ttl")
	}
}
