package collector

import (
	"encoding/json"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	bfdSubsystem = "bfd"
)

func init() {
	registerCollector(bfdSubsystem, enabledByDefault, NewBFDCollector)
}

type bfdCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewBFDCollector collects BFD metrics, implemented as per the Collector interface.
func NewBFDCollector(logger *slog.Logger) (Collector, error) {
	return &bfdCollector{logger: logger, descriptions: getBFDDesc()}, nil
}

func getBFDDesc() map[string]*prometheus.Desc {
	countLabels := []string{}
	peerLabels := []string{"local", "peer"}
	return map[string]*prometheus.Desc{
		"bfdPeerCount":  colPromDesc(bfdSubsystem, "peer_count", "Number of peers detected.", countLabels),
		"bfdPeerUptime": colPromDesc(bfdSubsystem, "peer_uptime", "Uptime of bfd peer in seconds", peerLabels),
		"bfdPeerState":  colPromDesc(bfdSubsystem, "peer_state", "State of the bfd peer (1 = Up, 0 = Down).", peerLabels),
	}
}

// Update implemented as per the Collector interface.
func (c *bfdCollector) Update(ch chan<- prometheus.Metric) error {
	cmd := "show bfd peers json"
	jsonBFDInterface, err := executeBFDCommand(cmd)
	if err != nil {
		return err
	}
	if err = processBFDPeers(ch, jsonBFDInterface, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(jsonBFDInterface), err)
	}
	return nil
}

func processBFDPeers(ch chan<- prometheus.Metric, jsonBFDInterface []byte, bfdDesc map[string]*prometheus.Desc) error {
	var bfdPeers []bfdPeer
	if err := json.Unmarshal(jsonBFDInterface, &bfdPeers); err != nil {
		return err
	}

	// metric is a count of the number of peers
	newGauge(ch, bfdDesc["bfdPeerCount"], float64(len(bfdPeers)))

	for _, p := range bfdPeers {

		labels := []string{p.Local, p.Peer}

		// get the uptime of the connection to the peer in seconds
		newGauge(ch, bfdDesc["bfdPeerUptime"], float64(p.Uptime), labels...)

		// state of connection to the bfd peer, up or down
		var bfdState float64
		if p.Status == "up" {
			bfdState = 1
		}
		newGauge(ch, bfdDesc["bfdPeerState"], bfdState, labels...)
	}
	return nil
}

type bfdPeer struct {
	Multihop               bool   `json:"multihop"`
	Peer                   string `json:"peer"`
	Local                  string `json:"local"`
	Vrf                    string `json:"vrf"`
	ID                     uint32 `json:"id"`
	RemoteID               uint32 `json:"remote-id"`
	Status                 string `json:"status"`
	Uptime                 uint64 `json:"uptime"`
	Diagnostic             string `json:"diagnostic"`
	RemoteDiagnostic       string `json:"remote-diagnostic"`
	ReceiveInterval        uint32 `json:"receive-interval"`
	TransmitInterval       uint32 `json:"transmit-interval"`
	EchoInterval           uint32 `json:"echo-interval"`
	RemoteReceiveInterval  uint32 `json:"remote-receive-interval"`
	RemoteTransmitInterval uint32 `json:"remote-transmit-interval"`
	RemoteEchoInterval     uint32 `json:"remote-echo-interval"`
}
