package collector

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pimSubsystem            = "pim"
	pimNeighborMetrixPrefix = "pim_neighbor"

	pimDesc map[string]*prometheus.Desc

	pimErrors      = []error{}
	totalPIMErrors = 0.0
)

// PIMCollector collects PIM metrics, implemented per prometheus.Collector interface.
type PIMCollector struct{}

// NewPIMCollector returns an empty PIMCollector struct.
func NewPIMCollector() *PIMCollector {
	return &PIMCollector{}
}

// Name of the collector.  Used to popuplate flag name.
func (*PIMCollector) Name() string {
	return pimSubsystem
}

// Help describes the metrics
func (*PIMCollector) Help() string {
	return "Collect PIM metrics"
}

// EnabledByDefault describes whether this collector is enabled by default.  Used to populate flag default
func (*PIMCollector) EnabledByDefault() bool {
	return false
}

// Describe implemented as per the prometheus.Collector interface
func (*PIMCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range getPimDesc() {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface
func (*PIMCollector) Collect(ch chan<- prometheus.Metric) {
	pimErrors = []error{}
	collectPIM(ch)
}

// CollectErrors returns what errors have been gathered.
func (*PIMCollector) CollectErrors() []error {
	return pimErrors
}

// CollectTotalErrors returns total errors.
func (*PIMCollector) CollectTotalErrors() float64 {
	return totalPIMErrors
}

func getPimDesc() map[string]*prometheus.Desc {
	if pimDesc != nil {
		return pimDesc
	}
	pimLabels := []string{"vrf"}
	pimInterfaceLabels := append(pimLabels, "iface")
	pimNeighborLabels := append(pimInterfaceLabels, "neighbor")
	pimDesc = map[string]*prometheus.Desc{
		"neighborCount": colPromDesc(pimSubsystem, "neighbors_count_total", "Number of neighbors detected", pimLabels),

		"upTime": colPromDesc(pimNeighborMetrixPrefix, "uptime_seconds", "How long has the peer ben up.", pimNeighborLabels),
	}
	return pimDesc
}

func collectPIM(ch chan<- prometheus.Metric) {
	pimErrors = []error{}
	totalPIMErrors = 0.0
	jsonPIMNeighbors, err := getPIMNeighbors()
	if err != nil {
		totalPIMErrors++
		pimErrors = append(pimErrors, fmt.Errorf("cannot get pim neighbors: %s", err))
	} else {
		if err := processPIMNeighbors(ch, jsonPIMNeighbors); err != nil {
			totalPIMErrors++
			pimErrors = append(pimErrors, err)
		}
	}
}

func getPIMNeighbors() ([]byte, error) {
	args := []string{"-c", "show ip pim vrf all neighbor json"}

	return execVtyshCommand(args...)
}

func processPIMNeighbors(ch chan<- prometheus.Metric, jsonPIMNeighbors []byte) error {
	var jsonMap map[string]json.RawMessage
	pimDesc := getPimDesc()
	if err := json.Unmarshal(jsonPIMNeighbors, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal pim neighbors json: %s", err)
	}
	for vrfName, vrfData := range jsonMap {
		neighborCount := 0.0
		var _tempvrfInstance map[string]json.RawMessage
		if err := json.Unmarshal(vrfData, &_tempvrfInstance); err != nil {
			return fmt.Errorf("cannot unmarshal VRF instance json: %s", err)
		}
		for ifaceName, ifaceData := range _tempvrfInstance {
			var neighbors map[string]pimNeighbor
			if err := json.Unmarshal(ifaceData, &neighbors); err != nil {
				return fmt.Errorf("cannot unmarshal neighbor json: %s", err)
			}
			for neighborIp, neighborData := range neighbors {
				neighborCount++
				if uptimeSec, err := parseHMS(neighborData.UpTime); err != nil {
					totalPIMErrors++
					pimErrors = append(pimErrors, fmt.Errorf("cannot parse neighbor uptime %s: %s", neighborData.UpTime, err))
				} else {
					// The labels are "vrf", "iface", "neighbor"
					neighborLabels := []string{strings.ToLower(vrfName), strings.ToLower(ifaceName), neighborIp}
					newGauge(ch, pimDesc["upTime"], float64(uptimeSec), neighborLabels...)
				}

			}
		}
		newGauge(ch, pimDesc["neighborCount"], neighborCount, vrfName)
	}
	return nil
}

func parseHMS(st string) (int, error) {
	var h, m, s int
	n, err := fmt.Sscanf(st, "%d:%d:%d", &h, &m, &s)
	if err != nil || n != 3 {
		return 0, err
	}
	return h*3600 + m*60 + s, nil
}

type pimNeighbor struct {
	Interface string
	Neighbor  string
	UpTime    string
}
