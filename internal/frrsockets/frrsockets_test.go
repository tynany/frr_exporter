package frrsockets

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteCmd(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "zebra_mock.vty")
	expected := "FRRouting 8.1 (localhost).\n"

	// Simple mock of FRR Zebra Unix socket
	go mockSocket(socketPath, expected)

	// Allow socket listener goroutine to settle
	time.Sleep(100 * time.Millisecond)

	if resp, err := executeCmd(socketPath, "show version", time.Second); err != nil {
		t.Fatalf("executeCmd returned error: %v\n", err)
	} else if string(resp) != expected {
		t.Fatalf("executeCmd expected '%s', got '%s'\n", expected, resp)
	}
}

// TestExecuteCmdWithLargeOutput tests ExecuteCmd when the command returns
// a large amount of output exceeding the hard-coded buffer size of 4096.
func TestExecuteCmdWithLargeOutput(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "bgp_mock.vty")

	command := "show a whole lot of data"
	expected := strings.Repeat("z", 5000)

	go mockSocket(socketPath, expected)

	// Allow socket listener goroutine to settle
	time.Sleep(100 * time.Millisecond)

	if resp, err := executeCmd(socketPath, command, time.Second); err != nil {
		t.Fatalf("executeCmd returned error: %v\n", err)
	} else if string(resp) != expected {
		t.Fatalf("executeCmd \n  expected '%s',\n       got '%s'\n",
			expected,
			resp)
	}
}

func mockSocket(socketPath string, socketData string) {
	// Simple mock of FRR Unix socket
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

	_, err = conn.Write([]byte(socketData + "\x00"))
	if err != nil {
		panic(err)
	}
}
