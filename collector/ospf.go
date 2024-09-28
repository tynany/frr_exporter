package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ospfSubsystem    = "ospf"
	frrOSPFInstances = kingpin.Flag("collector.ospf.instances", "Comma-separated list of instance IDs if using multiple OSPF instances").Default("").String()
)

func init() {
	registerCollector(ospfSubsystem, enabledByDefault, NewOSPFCollector)
}

type ospfCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
	instanceIDs  []int
}

// NewOSPFCollector  collects OSPF metrics, implemented as per the Collector interface.
func NewOSPFCollector(logger *slog.Logger) (Collector, error) {
	var instanceIDs []int
	if len(*frrOSPFInstances) > 0 {
		// FRR Exporter does not support multi-instance when using `vtysh` to interface with FRR
		// via the `--frr.vtysh` flag for the following reasons:
		//   * Invalid JSON is returned when OSPF commands are executed by `vtysh`. For example,
		//     `show ip ospf vrf all interface json` returns the concatenated JSON from each OSPF instance.
		//   * Vtysh does not support `vrf` and `instance` in the same commend. For example,
		//     `show ip ospf 1 vrf all interface json` is an invalid command.
		if *vtyshEnable {
			return nil, fmt.Errorf("cannot use --frr.vtysh with --collector.ospf.instances")
		}
		instances := strings.Split(*frrOSPFInstances, ",")
		for _, id := range instances {
			i, err := strconv.Atoi(id)
			if err != nil {
				return nil, fmt.Errorf("unable to parse instance ID %s: %w", id, err)
			}
			instanceIDs = append(instanceIDs, i)
		}
	}
	return &ospfCollector{logger: logger, instanceIDs: instanceIDs, descriptions: getOSPFDesc()}, nil
}

func getOSPFDesc() map[string]*prometheus.Desc {
	labels := []string{"vrf", "iface", "area"}
	if len(*frrOSPFInstances) > 0 {
		labels = append(labels, "instance")
	}
	return map[string]*prometheus.Desc{
		"ospfIfaceNeigh":    colPromDesc(ospfSubsystem, "neighbors", "Number of neighbors detected.", labels),
		"ospfIfaceNeighAdj": colPromDesc(ospfSubsystem, "neighbor_adjacencies", "Number of neighbor adjacencies formed.", labels),
	}
}

// Update implemented as per the Collector interface.
func (c *ospfCollector) Update(ch chan<- prometheus.Metric) error {
	cmd := "show ip ospf vrf all interface json"

	if len(c.instanceIDs) > 0 {
		for _, id := range c.instanceIDs {
			jsonOSPFInterface, err := executeOSPFMultiInstanceCommand(cmd, id)
			if err != nil {
				return err
			}

			if err = processOSPFInterface(ch, jsonOSPFInterface, c.descriptions, id); err != nil {
				return cmdOutputProcessError(cmd, string(jsonOSPFInterface), err)
			}
		}
		return nil
	}

	jsonOSPFInterface, err := executeOSPFCommand(cmd)
	if err != nil {
		return err
	}

	if err = processOSPFInterface(ch, jsonOSPFInterface, c.descriptions, 0); err != nil {
		return cmdOutputProcessError(cmd, string(jsonOSPFInterface), err)
	}
	return nil
}

func processOSPFInterface(ch chan<- prometheus.Metric, jsonOSPFInterface []byte, ospfDesc map[string]*prometheus.Desc, instanceID int) error {
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
		switch vrfName {
		case "ospfInstance":
			// Do nothing
		default:
			if err := json.Unmarshal(vrfData, &_tempvrfInstance); err != nil {
				return fmt.Errorf("cannot unmarshal VRF instance json: %s", err)
			}
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
					if !newIface.TimerPassiveIface {
						// The labels are "vrf", "newIface", "area"
						labels := []string{strings.ToLower(vrfName), interfaceKey, newIface.Area}
						ospfMetrics(ch, newIface, labels, ospfDesc, instanceID)
					}
				}
			default:
				// All other keys are interfaces.
				var iface ospfIface
				if err := json.Unmarshal(ospfInstanceVal, &iface); err != nil {
					return fmt.Errorf("cannot unmarshal interface json: %s", err)
				}
				if !iface.TimerPassiveIface {
					// The labels are "vrf", "iface", "area"
					labels := []string{strings.ToLower(vrfName), ospfInstanceKey, iface.Area}
					ospfMetrics(ch, iface, labels, ospfDesc, instanceID)
				}
			}
		}
	}
	return nil
}

func ospfMetrics(ch chan<- prometheus.Metric, iface ospfIface, labels []string, ospfDesc map[string]*prometheus.Desc, instanceID int) {
	if instanceID != 0 {
		labels = append(labels, strconv.Itoa(instanceID))
	}
	newGauge(ch, ospfDesc["ospfIfaceNeigh"], float64(iface.NbrCount), labels...)
	newGauge(ch, ospfDesc["ospfIfaceNeighAdj"], float64(iface.NbrAdjacentCount), labels...)
}

type ospfIface struct {
	NbrCount          uint32
	NbrAdjacentCount  uint32
	Area              string
	TimerPassiveIface bool
}
