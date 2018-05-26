package collector

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ospfSubsystem = "ospf"

	ospfIfaceLabels = []string{"vrf", "iface", "area"}

	ospfIfaceNeigh    = prometheus.NewDesc(prometheus.BuildFQName(namespace, ospfSubsystem, "neighbors"), "Number of neighbors deteceted.", ospfIfaceLabels, nil)
	ospfIfaceNeighAdj = prometheus.NewDesc(prometheus.BuildFQName(namespace, ospfSubsystem, "neighbor_adjacencies"), "Number of neighbor adjacencies formed.", ospfIfaceLabels, nil)
)

// OSPFCollector collects OSPF metrics from "vtysh -c 'show ip ospf vrf all interface json'"
type OSPFCollector struct {
	Errors int
}

func (*OSPFCollector) newCollector() Collector {
	return &OSPFCollector{}
}

// Describes the metrics.
func (*OSPFCollector) desc(ch chan<- *prometheus.Desc) {
	ch <- ospfIfaceNeigh
	ch <- ospfIfaceNeighAdj
}

// Name of the collector. Used to populate flag name.
func (*OSPFCollector) Name() string {
	return ospfSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*OSPFCollector) Help() string {
	return "Collect OSPF Metrics."
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*OSPFCollector) EnabledByDefault() bool {
	return true
}

func (c *OSPFCollector) scrape(ch chan<- prometheus.Metric) error {
	jsonOSPFInterface, err := getOSPFSummary()
	if err != nil {
		return fmt.Errorf("cannot get ospf interface summary: %s", err)
	}

	// Unfortunately, the 'show ip ospf vrf all interface json' JSON  output is poorly structured. Instead
	// of all interfaces being in a list, each interface is added as a key on the same level of vrfName and
	// vrfId. As such, we have to loop through each key and apply logic to determine whether the key is an
	// interface.
	var jsonMap map[string]json.RawMessage
	if err = json.Unmarshal(jsonOSPFInterface, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal ospf interface json: %s", err)
	}

	for vrfName, vrfData := range jsonMap {
		var _tempvrfInstance map[string]json.RawMessage
		if err := json.Unmarshal(vrfData, &_tempvrfInstance); err != nil {
			return fmt.Errorf("cannot unmarshal VRF instance json: %s", err)
		}

		for ospfInstanceKey, ospfInstanceVal := range _tempvrfInstance {
			switch ospfInstanceKey {
			case "vrfName", "vrfId":
				// Do nothing as we do not need the value of these keys.
			default:
				// All other keys are interfaces.
				var iface ospfIface
				if err := json.Unmarshal(ospfInstanceVal, &iface); err != nil {
					return fmt.Errorf("cannot unmarshal interface json: %s", err)
				}
				// The labels are "vrf", "iface", "area"
				labels := []string{vrfName, ospfInstanceKey, iface.Area}
				ch <- prometheus.MustNewConstMetric(ospfIfaceNeigh, prometheus.GaugeValue, float64(iface.NbrCount), labels...)
				ch <- prometheus.MustNewConstMetric(ospfIfaceNeighAdj, prometheus.GaugeValue, float64(iface.NbrAdjacentCount), labels...)
			}
		}
	}
	return nil
}

type ospfIface struct {
	NbrCount         int
	NbrAdjacentCount int
	Area             string
}

func getOSPFSummary() ([]byte, error) {
	args := []string{"-c", "show ip ospf vrf all interface json"}
	output, err := exec.Command(*frrVTYSHPath, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}
