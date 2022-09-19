package frrsockets

import (
	"encoding/json"
	"fmt"
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

		_, _ = conn.Write([]byte(expected + "\x00"))
	}()

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

	command := "show a whole lot of summary json"

	// create a structure mirroring the output of
	// "show bgp vrf all ipv4 unicast summary json"

	defaultSummaryDataResult := make(map[string]bgpSummaryData)

	defaultSummaryDataResult["default"] = bgpSummaryData{
		RouterId:        "10.0.0.254",
		As:              4100001254,
		Vrfid:           0,
		VrfName:         "default",
		TableVersion:    0,
		RibCount:        17,
		RibMemory:       3128,
		PeerCount:       32,
		PeerMemory:      23705856,
		PeerGroupCount:  2,
		PeerGroupMemory: 128,
		Peers:           make(map[string]peerData),
		FailedPeers:     0,
		DisplayedPeers:  32,
		TotalPeers:      32,
		DynamicPeers:    0,
		BestPath: bestPath{
			MultipathRelax: true,
		},
	}

	// add 32 peers which will produce a large amount of output exceeding the
	// internal buffer size in ExecuteCmd
	for i := 0; i < 32; i++ {
		peer := peerData{
			Hostname:                   fmt.Sprintf("host%d", i),
			RemoteAs:                   int64(4100001000 + i),
			LocalAs:                    4100001254,
			Version:                    4,
			MsgRcvd:                    100,
			MsgSent:                    1000,
			TableVersion:               0,
			Outq:                       0,
			Inq:                        0,
			PeerUptime:                 "06:58:46",
			PeerUptimeMsec:             25126000,
			PeerUptimeEstablishedEpoch: 1663578929,
			PfxRcd:                     4,
			PfxSnt:                     6,
			State:                      "Established",
			PeerState:                  "OK",
			ConnectionsEstablished:     1,
			ConnectionsDropped:         0,
			IdType:                     "interface",
		}
		defaultSummaryDataResult["default"].Peers[fmt.Sprintf("int%d", i)] = peer
	}

	expected, err := json.Marshal(defaultSummaryDataResult)
	if err != nil {
		panic(err)
	}

	expectedStr := string(expected)

	// Simple mock of FRR Unix socket
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

		_, err = conn.Write([]byte(expectedStr + "\x00"))
		if err != nil {
			panic(err)
		}
	}()

	// Allow socket listener goroutine to settle
	time.Sleep(100 * time.Millisecond)

	if resp, err := executeCmd(socketPath, command, time.Second); err != nil {
		t.Fatalf("executeCmd returned error: %v\n", err)
	} else if string(resp) != expectedStr {
		t.Fatalf("executeCmd \n  expected '%s',\n       got '%s'\n",
			expected,
			resp)
	}
}

type peerData struct {
	Hostname                   string `json:"hostname"`
	RemoteAs                   int64  `json:"remoteAs"`
	LocalAs                    int64  `json:"localAs"`
	Version                    int    `json:"version"`
	MsgRcvd                    int    `json:"msgRcvd"`
	MsgSent                    int    `json:"msgSent"`
	TableVersion               int    `json:"tableVersion"`
	Outq                       int    `json:"outq"`
	Inq                        int    `json:"inq"`
	PeerUptime                 string `json:"peerUptime"`
	PeerUptimeMsec             int    `json:"peerUptimeMsec"`
	PeerUptimeEstablishedEpoch int    `json:"peerUptimeEstablishedEpoch"`
	PfxRcd                     int    `json:"pfxRcd"`
	PfxSnt                     int    `json:"pfxSnt"`
	State                      string `json:"state"`
	PeerState                  string `json:"peerState"`
	ConnectionsEstablished     int    `json:"connectionsEstablished"`
	ConnectionsDropped         int    `json:"connectionsDropped"`
	IdType                     string `json:"idType"`
}

type bestPath struct {
	MultipathRelax bool `json:"multiPathRelax"`
}

type bgpSummaryData struct {
	RouterId        string              `json:"routerId"`
	As              int64               `json:"as"`
	Vrfid           int                 `json:"vrfId"`
	VrfName         string              `json:"vrfName"`
	TableVersion    int                 `json:"tableVersion"`
	RibCount        int                 `json:"ribCount"`
	RibMemory       int                 `json:"ribMemory"`
	PeerCount       int                 `json:"peerCount"`
	PeerMemory      int64               `json:"peerMemory"`
	PeerGroupCount  int                 `json:"peerGroupCount"`
	PeerGroupMemory int64               `json:"peerGroupMemory"`
	Peers           map[string]peerData `json:"peers"`
	FailedPeers     int                 `json:"failedPeers"`
	DisplayedPeers  int                 `json:"displayedPeers"`
	TotalPeers      int                 `json:"totalPeers"`
	DynamicPeers    int                 `json:"dynamicPeers"`
	BestPath        bestPath            `json:"bestPath"`
}
