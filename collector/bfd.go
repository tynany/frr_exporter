package collector

import (
	"encoding/json"
	"fmt"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	bfdSubsystem = "bfd"
)

func init() {
	registerCollector(bfdSubsystem, enabledByDefault, NewBFDCollector)
}

type bfdCollector struct {
	logger       log.Logger
	descriptions map[string]*prometheus.Desc
}

// NewBFDCollector collects BFD metrics, implemented as per the Collector interface.
func NewBFDCollector(logger log.Logger) (Collector, error) {
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
	jsonBFDInterface, err := getBFDInterface()
	if err != nil {
		return fmt.Errorf("cannot get bfd peers summary: %s", err)
	} else {
		if err = processBFDPeers(ch, jsonBFDInterface, c.descriptions); err != nil {
			return err
		}
	}
	return nil
}

func getBFDInterface() ([]byte, error) {
	return execVtyshCommand("-c", "show bfd peers json")
}

func processBFDPeers(ch chan<- prometheus.Metric, jsonBFDInterface []byte, bfdDesc map[string]*prometheus.Desc) error {
	var bfdPeers []bfdPeer
	if err := json.Unmarshal(jsonBFDInterface, &bfdPeers); err != nil {
		return fmt.Errorf("cannot unmarshal bfd peers json: %s", err)
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
	ID                     int    `json:"id"`
	RemoteID               int    `json:"remote-id"`
	Status                 string `json:"status"`
	Uptime                 int    `json:"uptime"`
	Diagnostic             string `json:"diagnostic"`
	RemoteDiagnostic       string `json:"remote-diagnostic"`
	ReceiveInterval        int    `json:"receive-interval"`
	TransmitInterval       int    `json:"transmit-interval"`
	EchoInterval           int    `json:"echo-interval"`
	RemoteReceiveInterval  int    `json:"remote-receive-interval"`
	RemoteTransmitInterval int    `json:"remote-transmit-interval"`
	RemoteEchoInterval     int    `json:"remote-echo-interval"`
}
