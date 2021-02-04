package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ospfSubsystem = "ospf"

	ospfIfaceLabels = []string{"vrf", "iface", "area"}
	ospfDesc        = map[string]*prometheus.Desc{
		"ospfIfaceNeigh":    colPromDesc(ospfSubsystem, "neighbors", "Number of neighbors detected.", ospfIfaceLabels),
		"ospfIfaceNeighAdj": colPromDesc(ospfSubsystem, "neighbor_adjacencies", "Number of neighbor adjacencies formed.", ospfIfaceLabels),
	}
	ospfErrors      = []error{}
	totalOSPFErrors = 0.0
)

// OSPFCollector collects OSPF metrics, implemented as per prometheus.Collector interface.
type OSPFCollector struct{}

// NewOSPFCollector returns a OSPFCollector struct.
func NewOSPFCollector() *OSPFCollector {
	return &OSPFCollector{}
}

// Name of the collector. Used to populate flag name.
func (*OSPFCollector) Name() string {
	return ospfSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*OSPFCollector) Help() string {
	return "Collect OSPF Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*OSPFCollector) EnabledByDefault() bool {
	return true
}

// Describe implemented as per the prometheus.Collector interface.
func (*OSPFCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range ospfDesc {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *OSPFCollector) Collect(ch chan<- prometheus.Metric) {
	jsonOSPFInterface, err := getOSPFInterface()
	if err != nil {
		totalOSPFErrors++
		ospfErrors = append(ospfErrors, fmt.Errorf("cannot get ospf interface summary: %s", err))
	} else {
		if err = processOSPFInterface(ch, jsonOSPFInterface); err != nil {
			totalOSPFErrors++
			ospfErrors = append(ospfErrors, fmt.Errorf("%s", err))
		}
	}
}

// CollectErrors returns what errors have been gathered.
func (*OSPFCollector) CollectErrors() []error {
	return ospfErrors
}

// CollectTotalErrors returns total errors.
func (*OSPFCollector) CollectTotalErrors() float64 {
	return totalOSPFErrors
}

func getOSPFInterface() ([]byte, error) {
	args := []string{"-c", "show ip ospf vrf all interface json"}
	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, vtyshPath, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func processOSPFInterface(ch chan<- prometheus.Metric, jsonOSPFInterface []byte) error {
	// Unfortunately, the 'show ip ospf vrf all interface json' JSON  output is poorly structured. Instead
	// of all interfaces being in a list, each interface is added as a key on the same level of vrfName and
	// vrfId. As such, we have to loop through each key and apply logic to determine whether the key is an
	// interface.
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(jsonOSPFInterface, &jsonMap); err != nil {
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
			case "interfaces":
				var _tempInterfaceInstance map[string]json.RawMessage
				if err := json.Unmarshal(ospfInstanceVal, &_tempInterfaceInstance); err != nil {
					return fmt.Errorf("cannot unmarshal VRF instance json: %s", err)
				}
				for interfaceKey, interfaceValue := range _tempInterfaceInstance {
					var newIface ospfIface
					if err := json.Unmarshal(interfaceValue, &newIface); err != nil {
						return fmt.Errorf("cannot unmarshal interface json: %s", err)
					}
					// The labels are "vrf", "newIface", "area"
					labels := []string{strings.ToLower(vrfName), interfaceKey, newIface.Area}
					newGauge(ch, ospfDesc["ospfIfaceNeigh"], newIface.NbrCount, labels...)
					newGauge(ch, ospfDesc["ospfIfaceNeighAdj"], newIface.NbrAdjacentCount, labels...)
				}
			default:
				// All other keys are interfaces.
				var iface ospfIface
				if err := json.Unmarshal(ospfInstanceVal, &iface); err != nil {
					return fmt.Errorf("cannot unmarshal interface json: %s", err)
				}
				// The labels are "vrf", "iface", "area"
				labels := []string{strings.ToLower(vrfName), ospfInstanceKey, iface.Area}
				newGauge(ch, ospfDesc["ospfIfaceNeigh"], iface.NbrCount, labels...)
				newGauge(ch, ospfDesc["ospfIfaceNeighAdj"], iface.NbrAdjacentCount, labels...)
			}
		}
	}
	return nil
}

type ospfIface struct {
	NbrCount         float64
	NbrAdjacentCount float64
	Area             string
}
