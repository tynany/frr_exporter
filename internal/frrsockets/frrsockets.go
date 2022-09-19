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
	return executeCmd(filepath.Join(c.dirPath, "bgpd.vty"), cmd, c.timeout)
}

func (c Connection) ExecOSPFCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Join(c.dirPath, "ospfd.vty"), cmd, c.timeout)
}

func (c Connection) ExecOSPFMultiInstanceCmd(cmd string, instanceID int) ([]byte, error) {
	return executeCmd(filepath.Join(c.dirPath, fmt.Sprintf("ospfd-%d.vty", instanceID)), cmd, c.timeout)
}

func (c Connection) ExecPIMCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Join(c.dirPath, "pimd.vty"), cmd, c.timeout)
}

func (c Connection) ExecVRRPCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Join(c.dirPath, "vrrpd.vty"), cmd, c.timeout)
}

func (c Connection) ExecZebraCmd(cmd string) ([]byte, error) {
	return executeCmd(filepath.Join(c.dirPath, "zebra.vty"), cmd, c.timeout)
}

func executeCmd(socketPath, cmd string, timeout time.Duration) ([]byte, error) {
	var response bytes.Buffer

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Net: "unix", Name: socketPath})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	// frr vty sockets expect command to be null-terminated
	if _, err = conn.Write([]byte(cmd + "\x00")); err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return response.Bytes(), err
		}

		response.Write(buf[:n])

		// frr signals the end of a response with a null character
		if n > 0 && buf[n-1] == 0 {
			return bytes.TrimRight(response.Bytes(), "\x00"), nil
		}
	}
}
