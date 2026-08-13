package core

import (
	"io"
	"net"
	"os"
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

func TestEvalSAVE(t *testing.T) {
	withTempDumpFile(t)
	setKey("tosave", "value")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalSAVE(nil, server); err != nil {
			t.Errorf("evalSAVE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != "+OK\r\n" {
		t.Fatalf("unexpected response: %q", resp)
	}

	if _, err := os.Stat(dumpFile); err != nil {
		t.Fatalf("expected dump file to exist after SAVE: %v", err)
	}
}

func TestEvalSAVEWithArgsErrors(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- evalSAVE([]string{"unexpected"}, server)
	}()

	if err := <-errCh; err == nil {
		t.Fatalf("expected an error when SAVE is given arguments")
	}
}

func TestEvalKEYS(t *testing.T) {
	resetStore(t)
	setKey("foo", "value1")
	setKey("foobar", "value2")
	setKey("bar", "value3")
	setKey("baz", "value4")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalKEYS([]string{"foo*"}, server); err != nil {
			t.Errorf("evalKEYS error: %v", err)
		}
	}()

	resp := readResponse(t, client)
	// Should contain foo and foobar
	respStr := string(resp)
	if !(contains(respStr, "foo") && contains(respStr, "foobar")) {
		t.Fatalf("expected 'foo' and 'foobar' in response, got: %q", resp)
	}
	if contains(respStr, "bar") && !contains(respStr, "foobar") {
		t.Fatalf("expected 'bar' to not be in response (unless as part of foobar), got: %q", resp)
	}
}

func TestEvalKEYSMatchAll(t *testing.T) {
	resetStore(t)
	setKey("a", "1")
	setKey("b", "2")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalKEYS([]string{"*"}, server); err != nil {
			t.Errorf("evalKEYS error: %v", err)
		}
	}()

	resp := readResponse(t, client)
	respStr := string(resp)
	if !(contains(respStr, "a") && contains(respStr, "b")) {
		t.Fatalf("expected 'a' and 'b' in response, got: %q", resp)
	}
}

func TestEvalKEYSNoMatch(t *testing.T) {
	resetStore(t)
	setKey("foo", "value")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalKEYS([]string{"nomatch*"}, server); err != nil {
			t.Errorf("evalKEYS error: %v", err)
		}
	}()

	resp := readResponse(t, client)
	// Should be an empty array
	if string(resp) != "*0\r\n" {
		t.Fatalf("expected empty array response, got: %q", resp)
	}
}

func TestEvalDBSIZE(t *testing.T) {
	resetStore(t)
	setKey("key1", "value1")
	setKey("key2", "value2")
	setKey("key3", "value3")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalDBSIZE([]string{}, server); err != nil {
			t.Errorf("evalDBSIZE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":3\r\n" {
		t.Fatalf("expected ':3\\r\\n', got %q", resp)
	}
}

func TestEvalDBSIZEEmpty(t *testing.T) {
	resetStore(t)

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalDBSIZE([]string{}, server); err != nil {
			t.Errorf("evalDBSIZE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":0\r\n" {
		t.Fatalf("expected ':0\\r\\n', got %q", resp)
	}
}

func TestEvalDBSIZEIgnoresExpiredKeys(t *testing.T) {
	resetStore(t)
	setKey("persistent", "value")
	setKeyWithTTL("expiring", "value", 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		if err := evalDBSIZE([]string{}, server); err != nil {
			t.Errorf("evalDBSIZE error: %v", err)
		}
	}()

	if resp := readResponse(t, client); string(resp) != ":1\r\n" {
		t.Fatalf("expected ':1\\r\\n' (only persistent key), got %q", resp)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
