package frrsockets

import (
	"bytes"
	"fmt"
	"net"
	"path/filepath"
	"time"
)

type Connection struct {
	dirPath string
	timeout time.Duration
}

func NewConnection(dirPath string, timeout time.Duration) *Connection {
	return &Connection{dirPath: dirPath, timeout: timeout}
}

func (c Connection) ExecBGPCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+"/bgpd.vty"), cmd, c.timeout)
}

func (c Connection) ExecOSPFCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+"/ospfd.vty"), cmd, c.timeout)
}

func (c Connection) ExecOSPFMultiInstanceCmd(cmd string, instanceID int) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+fmt.Sprintf("/ospfd-%d.vty", instanceID)), cmd, c.timeout)
}

func (c Connection) ExecPIMCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+"/pimd.vty"), cmd, c.timeout)
}

func (c Connection) ExecZebraCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+"/zebra.vty"), cmd, c.timeout)
}

func (c Connection) ExecVRRPCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Clean(c.dirPath+"/vrrpd.vty"), cmd, c.timeout)
}

func executeCmd(socketPath, cmd string, timeout time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	addr := net.UnixAddr{Name: socketPath, Net: "unix"}

	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return buf.Bytes(), err
	}

	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return buf.Bytes(), err
	}

	// frr sockets expect each command to end with \0
	_, err = conn.Write([]byte(fmt.Sprintf("%s\000", cmd)))
	if err != nil {
		return buf.Bytes(), err
	}

	for {
		b := make([]byte, 1024)
		_, err := conn.Read(b)
		if err != nil {
			return buf.Bytes(), err
		}
		// frr signals the end of a response with \x00
		if bytes.HasSuffix(b, []byte("\x00")) {
			buf.Write(bytes.Trim(b, "\x00"))
			conn.Close()
			return buf.Bytes(), nil
		}
		buf.Write(b)
	}
}
