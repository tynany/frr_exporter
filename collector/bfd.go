package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	bfdSubsystem = "bfd"

	bfdCountLabels = []string{}
	bfdPeerLabels  = []string{"local", "peer"}
	bfdDesc        = map[string]*prometheus.Desc{
		"bfdPeerCount":  colPromDesc(bfdSubsystem, "peer_count", "Number of peers detected.", bfdCountLabels),
		"bfdPeerUptime": colPromDesc(bfdSubsystem, "peer_uptime", "Uptime of bfd peer", bfdPeerLabels),
		"bfdPeerState":  colPromDesc(bfdSubsystem, "peer_state", "State of the bfd peer", bfdPeerLabels),
	}
	bfdErrors      = []error{}
	totalBFDErrors = 0.0
)

// BFDCollector collects BFD metrics, implemented as per prometheus.Collector interface.
type BFDCollector struct{}

// NewBFDCollector returns a BFDCollector struct.
func NewBFDCollector() *BFDCollector {
	return &BFDCollector{}
}

// Name of the collector. Used to populate flag name.
func (*BFDCollector) Name() string {
	return bfdSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*BFDCollector) Help() string {
	return "Collect BFD Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*BFDCollector) EnabledByDefault() bool {
	return true
}

// Describe implemented as per the prometheus.Collector interface.
func (*BFDCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range bfdDesc {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *BFDCollector) Collect(ch chan<- prometheus.Metric) {
	jsonBFDInterface, err := getBFDInterface()
	if err != nil {
		totalBFDErrors++
		bfdErrors = append(bfdErrors, fmt.Errorf("cannot get bfd peers summary: %s", err))
	} else {
		if err = processBFDPeers(ch, jsonBFDInterface); err != nil {
			totalBFDErrors++
			bfdErrors = append(bfdErrors, fmt.Errorf("%s", err))
		}
	}
}

// CollectErrors returns what errors have been gathered.
func (*BFDCollector) CollectErrors() []error {
	return bfdErrors
}

// CollectTotalErrors returns total errors.
func (*BFDCollector) CollectTotalErrors() float64 {
	return totalBFDErrors
}

func getBFDInterface() ([]byte, error) {
	args := []string{"-c", "show bfd peers json"}
	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, vtyshPath, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func processBFDPeers(ch chan<- prometheus.Metric, jsonBFDInterface []byte) error {
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
