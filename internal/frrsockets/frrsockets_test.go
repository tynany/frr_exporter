package frrsockets

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteCmd(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "zebra_mock.vty")
	expected := "FRRouting 8.1 (localhost).\n"

	// Simple mock of FRR Zebra Unix socket
	go func() {
		l, err := net.Listen("unix", socketPath)
		if err != nil {
			panic(err)
		}
		defer os.Remove(socketPath)
		defer l.Close()

		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		cmd := make([]byte, 1024)
		if _, err := conn.Read(cmd); err != nil {
			panic(err)
		}

		conn.Write([]byte(expected + "\x00"))
	}()

	// Allow socket listener goroutine to settle
	time.Sleep(100 * time.Millisecond)

	if resp, err := executeCmd(socketPath, "show version", time.Second); err != nil {
		t.Fatalf("executeCmd returned error: %v\n", err)
	} else if string(resp) != expected {
		t.Fatalf("executeCmd expected '%s', got '%s'\n", expected, resp)
	}
}
