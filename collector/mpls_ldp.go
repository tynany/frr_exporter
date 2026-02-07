package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	mplsLdpSubsystem = "mpls_ldp"
)

func init() {
	registerCollector(mplsLdpSubsystem, enabledByDefault, NewMPLSLDPCollector)
}

type mplsLDPCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

func NewMPLSLDPCollector(logger *slog.Logger) (Collector, error) {
	return &mplsLDPCollector{logger: logger, descriptions: getMPLSLDPDesc()}, nil
}

func getMPLSLDPDesc() map[string]*prometheus.Desc {
	bindingsLabels := []string{"address_family"}
	igpSyncLabels := []string{"interface", "peer_ldp_id"}
	interfaceLabels := []string{"name", "address_family"}
	neighborLabels := []string{"address_family", "neighbor_id"}
	discoveryLabels := []string{"address_family", "neighbor_id", "interface", "type"}

	return map[string]*prometheus.Desc{
		"bindingCount":            colPromDesc(mplsLdpSubsystem, "binding_count", "Number of MPLS LDP bindings.", bindingsLabels),
		"igpSyncState":            colPromDesc(mplsLdpSubsystem, "igp_sync_state", "State of MPLS LDP IGP sync (1=Ready/Complete, 0=Not Complete).", igpSyncLabels),
		"interfaceState":          colPromDesc(mplsLdpSubsystem, "interface_state", "State of MPLS LDP interface (1=Active, 0=Inactive).", interfaceLabels),
		"interfaceHelloInterval":  colPromDesc(mplsLdpSubsystem, "interface_hello_interval_seconds", "Hello interval for the interface.", interfaceLabels),
		"interfaceHelloHoldtime":  colPromDesc(mplsLdpSubsystem, "interface_hello_holdtime_seconds", "Hello holdtime for the interface.", interfaceLabels),
		"interfaceAdjacencyCount": colPromDesc(mplsLdpSubsystem, "interface_adjacency_count", "Number of adjacencies on the interface.", interfaceLabels),
		"neighborState":           colPromDesc(mplsLdpSubsystem, "neighbor_state", "State of MPLS LDP neighbor (1=Operational, 0=Other).", neighborLabels),
		"neighborUptime":          colPromDesc(mplsLdpSubsystem, "neighbor_uptime_seconds", "Uptime of MPLS LDP neighbor in seconds.", neighborLabels),
		"discoveryAdjacencyCount": colPromDesc(mplsLdpSubsystem, "discovery_adjacency_count", "Number of discovery adjacencies.", discoveryLabels),
	}
}

func (c *mplsLDPCollector) Update(ch chan<- prometheus.Metric) error {
	// 1. Bindings
	if err := c.collectBindings(ch); err != nil {
		return err
	}
	// 2. IGP Sync
	if err := c.collectIGPSync(ch); err != nil {
		return err
	}
	// 3. Interface
	if err := c.collectInterface(ch); err != nil {
		return err
	}
	// 4. Neighbor
	if err := c.collectNeighbor(ch); err != nil {
		return err
	}
	// 5. Discovery
	if err := c.collectDiscovery(ch); err != nil {
		return err
	}
	return nil
}

// ----------------------------------------------------------------------
// Bindings
// ----------------------------------------------------------------------

type mplsLdpBindings struct {
	Bindings []struct {
		AddressFamily string `json:"addressFamily"`
	} `json:"bindings"`
}

func (c *mplsLDPCollector) collectBindings(ch chan<- prometheus.Metric) error {
	cmd := "show mpls ldp binding json"
	output, err := executeMPLSLDPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processBindings(ch, output, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(output), err)
	}
	return nil
}

func processBindings(ch chan<- prometheus.Metric, output []byte, descs map[string]*prometheus.Desc) error {
	var data mplsLdpBindings
	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	counts := make(map[string]float64)
	for _, b := range data.Bindings {
		counts[b.AddressFamily]++
	}

	for af, count := range counts {
		newGauge(ch, descs["bindingCount"], count, af)
	}
	return nil
}

// ----------------------------------------------------------------------
// IGP Sync
// ----------------------------------------------------------------------

type mplsLdpIGPSync struct {
	State     string `json:"state"`
	WaitTime  int    `json:"waitTime"`
	PeerLdpID string `json:"peerLdpId"`
}

type mplsLdpIGPSyncOutput map[string]mplsLdpIGPSync

func (c *mplsLDPCollector) collectIGPSync(ch chan<- prometheus.Metric) error {
	cmd := "show mpls ldp igp-sync json"
	output, err := executeMPLSLDPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processIGPSync(ch, output, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(output), err)
	}
	return nil
}

func processIGPSync(ch chan<- prometheus.Metric, output []byte, descs map[string]*prometheus.Desc) error {
	var data mplsLdpIGPSyncOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	for iface, info := range data {
		stateVal := 0.0
		if !strings.Contains(strings.ToLower(info.State), "notcomplete") {
			stateVal = 1.0
		}
		newGauge(ch, descs["igpSyncState"], stateVal, iface, info.PeerLdpID)
	}
	return nil
}

// ----------------------------------------------------------------------
// Interface
// ----------------------------------------------------------------------

type mplsLdpInterface struct {
	Name           string  `json:"name"`
	AddressFamily  string  `json:"addressFamily"`
	State          string  `json:"state"`
	UpTime         string  `json:"upTime"`
	HelloInterval  float64 `json:"helloInterval"`
	HelloHoldtime  float64 `json:"helloHoldtime"`
	AdjacencyCount float64 `json:"adjacencyCount"`
}

type mplsLdpInterfaceOutput map[string]mplsLdpInterface

func (c *mplsLDPCollector) collectInterface(ch chan<- prometheus.Metric) error {
	cmd := "show mpls ldp interface json"
	output, err := executeMPLSLDPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processInterface(ch, output, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(output), err)
	}
	return nil
}

func processInterface(ch chan<- prometheus.Metric, output []byte, descs map[string]*prometheus.Desc) error {
	var data mplsLdpInterfaceOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	for _, info := range data {
		stateVal := 0.0
		if strings.EqualFold(info.State, "active") {
			stateVal = 1.0
		}
		newGauge(ch, descs["interfaceState"], stateVal, info.Name, info.AddressFamily)
		newGauge(ch, descs["interfaceHelloInterval"], info.HelloInterval, info.Name, info.AddressFamily)
		newGauge(ch, descs["interfaceHelloHoldtime"], info.HelloHoldtime, info.Name, info.AddressFamily)
		newGauge(ch, descs["interfaceAdjacencyCount"], info.AdjacencyCount, info.Name, info.AddressFamily)
	}
	return nil
}

// ----------------------------------------------------------------------
// Neighbor
// ----------------------------------------------------------------------

type mplsLdpNeighbor struct {
	AddressFamily string `json:"addressFamily"`
	NeighborID    string `json:"neighborId"`
	State         string `json:"state"`
	UpTime        string `json:"upTime"`
}

type mplsLdpNeighborOutput struct {
	Neighbors []mplsLdpNeighbor `json:"neighbors"`
}

func (c *mplsLDPCollector) collectNeighbor(ch chan<- prometheus.Metric) error {
	cmd := "show mpls ldp neighbor json"
	output, err := executeMPLSLDPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processNeighbor(ch, output, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(output), err)
	}
	return nil
}

func processNeighbor(ch chan<- prometheus.Metric, output []byte, descs map[string]*prometheus.Desc) error {
	var data mplsLdpNeighborOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	for _, n := range data.Neighbors {
		stateVal := 0.0
		if strings.EqualFold(n.State, "operational") {
			stateVal = 1.0
		}
		newGauge(ch, descs["neighborState"], stateVal, n.AddressFamily, n.NeighborID)

		uptimeSeconds, err := parseUptime(n.UpTime)
		if err == nil {
			newGauge(ch, descs["neighborUptime"], uptimeSeconds, n.AddressFamily, n.NeighborID)
		}
	}
	return nil
}

// ----------------------------------------------------------------------
// Discovery
// ----------------------------------------------------------------------

type mplsLdpDiscovery struct {
	AddressFamily string `json:"addressFamily"`
	NeighborID    string `json:"neighborId"`
	Type          string `json:"type"`
	Interface     string `json:"interface"`
}

type mplsLdpDiscoveryOutput struct {
	Adjacencies []mplsLdpDiscovery `json:"adjacencies"`
}

func (c *mplsLDPCollector) collectDiscovery(ch chan<- prometheus.Metric) error {
	cmd := "show mpls ldp discovery json"
	output, err := executeMPLSLDPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processDiscovery(ch, output, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(output), err)
	}
	return nil
}

func processDiscovery(ch chan<- prometheus.Metric, output []byte, descs map[string]*prometheus.Desc) error {
	var data mplsLdpDiscoveryOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	counts := make(map[string]float64)
	for _, d := range data.Adjacencies {
		key := strings.Join([]string{d.AddressFamily, d.NeighborID, d.Interface, d.Type}, "|")
		counts[key]++
	}

	for key, count := range counts {
		parts := strings.Split(key, "|")
		if len(parts) == 4 {
			newGauge(ch, descs["discoveryAdjacencyCount"], count, parts[0], parts[1], parts[2], parts[3])
		}
	}
	return nil
}

// Helper to parse uptime string "HH:MM:SS"
func parseUptime(uptime string) (float64, error) {
	parts := strings.Split(uptime, ":")
	if len(parts) == 3 {
		h, err1 := parseInt(parts[0])
		m, err2 := parseInt(parts[1])
		s, err3 := parseInt(parts[2])
		if err1 == nil && err2 == nil && err3 == nil {
			return float64(h*3600 + m*60 + s), nil
		}
	}
	return 0, fmt.Errorf("invalid uptime format: %s", uptime)
}

func parseInt(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}
