package core

import (
	"io"
	"net"
	"testing"
	"time"
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

func TestEvalTTLNoExpiry(t *testing.T) {
	setKey("persistent", "value")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalTTL([]string{"persistent"}, server); err != nil {
			t.Errorf("evalTTL error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":-1\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}
}

func TestEvalTTLMissingKey(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalTTL([]string{"does-not-exist"}, server); err != nil {
			t.Errorf("evalTTL error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":-2\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}
}

func TestEvalSETWithEX(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalSET([]string{"withex", "value", "EX", "100"}, server); err != nil {
			t.Errorf("evalSET error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != "+OK\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}

	if ttl := ttlSeconds("withex"); ttl <= 0 {
		t.Fatalf("expected positive ttl after SET ... EX, got %d", ttl)
	}
}

func TestEvalSETWithPX(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalSET([]string{"withpx", "value", "PX", "20"}, server); err != nil {
			t.Errorf("evalSET error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != "+OK\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}

	time.Sleep(30 * time.Millisecond)

	if _, ok := getKey("withpx"); ok {
		t.Fatalf("expected key to be gone after PX expiry")
	}
}

func TestEvalSETInvalidExpireSyntax(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- evalSET([]string{"bad", "value", "NOTAFLAG", "100"}, server)
	}()

	if err := <-errCh; err == nil {
		t.Fatalf("expected an error for invalid SET expire syntax")
	}
}
