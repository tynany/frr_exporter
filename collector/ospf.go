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
	logger                     *slog.Logger
	ospfIfaceDescriptions      map[string]*prometheus.Desc
	ospfDescriptions           map[string]*prometheus.Desc
	ospfNeighDescriptions      map[string]*prometheus.Desc
	ospfDataMaxAgeDescriptions map[string]*prometheus.Desc
	instanceIDs                []int
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
	return &ospfCollector{logger: logger, instanceIDs: instanceIDs, ospfIfaceDescriptions: getOSPFIfaceDesc(), ospfDescriptions: getOSPFDesc(), ospfNeighDescriptions: getOSPFNeighDesc(), ospfDataMaxAgeDescriptions: getOSPFDataMaxAgeDesc()}, nil
}

// Update satisfies Collector.
func (c *ospfCollector) Update(ch chan<- prometheus.Metric) error {
	steps := []struct {
		cmd       string
		desc      map[string]*prometheus.Desc
		processor func(chan<- prometheus.Metric, []byte, map[string]*prometheus.Desc, int) error
	}{
		{
			cmd:       "show ip ospf vrf all json",
			desc:      c.ospfDescriptions,
			processor: processOSPF,
		},
		{
			cmd:       "show ip ospf vrf all interface json",
			desc:      c.ospfIfaceDescriptions,
			processor: processOSPFInterface,
		},
		{
			cmd:       "show ip ospf vrf all neighbor json",
			desc:      c.ospfNeighDescriptions,
			processor: processOSPFNeigh,
		},
		{
			cmd:       "show ip ospf vrf all database max-age json",
			desc:      c.ospfDataMaxAgeDescriptions,
			processor: processOSPFDataMaxAge,
		},
	}

	for _, s := range steps {
		if err := c.update(ch, s.cmd, s.desc, s.processor); err != nil {
			return err
		}
	}
	return nil
}

func (c *ospfCollector) update(
	ch chan<- prometheus.Metric,
	cmd string,
	descriptions map[string]*prometheus.Desc,
	process func(chan<- prometheus.Metric, []byte, map[string]*prometheus.Desc, int) error,
) error {
	if len(c.instanceIDs) > 0 {
		for _, id := range c.instanceIDs {
			jsonBytes, err := executeOSPFMultiInstanceCommand(cmd, id)
			if err != nil {
				return err
			}
			if err := process(ch, jsonBytes, descriptions, id); err != nil {
				return cmdOutputProcessError(cmd, string(jsonBytes), err)
			}
		}
		return nil
	}

	jsonBytes, err := executeOSPFCommand(cmd)
	if err != nil {
		return err
	}
	if err := process(ch, jsonBytes, descriptions, 0); err != nil {
		return cmdOutputProcessError(cmd, string(jsonBytes), err)
	}
	return nil
}

func getOSPFIfaceDesc() map[string]*prometheus.Desc {
	labels := []string{"vrf", "iface", "area"}
	if len(*frrOSPFInstances) > 0 {
		labels = append(labels, "instance")
	}
	return map[string]*prometheus.Desc{
		"ospfIfaceNeigh":    colPromDesc(ospfSubsystem, "neighbors", "Number of neighbors detected.", labels),
		"ospfIfaceNeighAdj": colPromDesc(ospfSubsystem, "neighbor_adjacencies", "Number of neighbor adjacencies formed.", labels),
	}
}

func getOSPFDesc() map[string]*prometheus.Desc {
	routerLabels := []string{"vrf"}
	areaLabels := []string{"vrf", "area"}
	if len(*frrOSPFInstances) > 0 {
		routerLabels = append(routerLabels, "instance")
		areaLabels = append(areaLabels, "instance")
	}

	return map[string]*prometheus.Desc{
		"ospfLsaExternalCounter":   colPromDesc(ospfSubsystem, "lsa_external_counter", "Number of external LSAs.", routerLabels),
		"ospfLsaAsOpaqueCounter":   colPromDesc(ospfSubsystem, "lsa_as_opaque_counter", "Number of AS Opaque LSAs.", routerLabels),
		"ospfAreaLsaNumber":        colPromDesc(ospfSubsystem, "area_lsa_number", "Number of LSAs in the area.", areaLabels),
		"ospfAreaLsaNetworkNumber": colPromDesc(ospfSubsystem, "area_lsa_network_number", "Number of network LSAs in the area.", areaLabels),
		"ospfAreaLsaSummaryNumber": colPromDesc(ospfSubsystem, "area_lsa_summary_number", "Number of summary LSAs in the area.", areaLabels),
		"ospfAreaLsaAsbrNumber":    colPromDesc(ospfSubsystem, "area_lsa_asbr_number", "Number of ASBR LSAs in the area.", areaLabels),
		"ospfAreaLsaNssaNumber":    colPromDesc(ospfSubsystem, "area_lsa_nssa_number", "Number of NSSA LSAs in the area.", areaLabels),
	}
}

func getOSPFNeighDesc() map[string]*prometheus.Desc {
	var labels []string
	if len(*frrOSPFInstances) > 0 {
		labels = append(labels, "instance")
	}
	labels = append(labels, "vrf", "neighbor", "iface", "local_address", "remote_address")
	return map[string]*prometheus.Desc{
		"ospfNeighState": colPromDesc(ospfSubsystem, "neighbor_state", "OSPF neighbor state (1=Down, 2=Init, 3=2-Way, 4=ExStart, 5=Exchange, 6=Loading, 7=Full).", labels),
	}
}

func getOSPFDataMaxAgeDesc() map[string]*prometheus.Desc {
	var labels []string
	if len(*frrOSPFInstances) > 0 {
		labels = append(labels, "instance")
	}
	labels = append(labels, "vrf")
	return map[string]*prometheus.Desc{
		"ospfDataMaxAge": colPromDesc(ospfSubsystem, "data_ls_max_age", "Amount of link state max age entries.", labels),
	}
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
						ospfIfaceMetrics(ch, newIface, labels, ospfDesc, instanceID)
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
					ospfIfaceMetrics(ch, iface, labels, ospfDesc, instanceID)
				}
			}
		}
	}
	return nil
}

func ospfIfaceMetrics(ch chan<- prometheus.Metric, iface ospfIface, labels []string, ospfDesc map[string]*prometheus.Desc, instanceID int) {
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

func processOSPF(ch chan<- prometheus.Metric, jsonOSPF []byte, ospfDesc map[string]*prometheus.Desc, instanceID int) error {
	var all map[string]ospfInstance
	if err := json.Unmarshal(jsonOSPF, &all); err != nil {
		return fmt.Errorf("cannot unmarshal ospf json: %w", err)
	}

	for vrfName, vrfData := range all {
		ospfMetrics(ch, vrfData, vrfName, ospfDesc, instanceID)
	}
	return nil
}

func ospfMetrics(ch chan<- prometheus.Metric, ospfData ospfInstance, vrfName string, ospfDesc map[string]*prometheus.Desc, instanceID int) {
	routerLabels := []string{strings.ToLower(vrfName)}
	if instanceID != 0 {
		routerLabels = append(routerLabels, strconv.Itoa(instanceID))
	}
	newGauge(ch, ospfDesc["ospfLsaExternalCounter"], float64(ospfData.LsaExternalCounter), routerLabels...)
	newGauge(ch, ospfDesc["ospfLsaAsOpaqueCounter"], float64(ospfData.LsaAsopaqueCounter), routerLabels...)

	for areaName, area := range ospfData.Areas {
		areaLabels := []string{strings.ToLower(vrfName), areaName}
		if instanceID != 0 {
			areaLabels = append(areaLabels, strconv.Itoa(instanceID))
		}
		newGauge(ch, ospfDesc["ospfAreaLsaNumber"], float64(area.LsaNumber), areaLabels...)
		newGauge(ch, ospfDesc["ospfAreaLsaNetworkNumber"], float64(area.LsaNetworkNumber), areaLabels...)
		newGauge(ch, ospfDesc["ospfAreaLsaSummaryNumber"], float64(area.LsaSummaryNumber), areaLabels...)
		newGauge(ch, ospfDesc["ospfAreaLsaAsbrNumber"], float64(area.LsaAsbrNumber), areaLabels...)
		newGauge(ch, ospfDesc["ospfAreaLsaNssaNumber"], float64(area.LsaNssaNumber), areaLabels...)
	}
}

type ospfInstance struct {
	LsaExternalCounter uint32
	LsaAsopaqueCounter uint32
	Areas              map[string]ospfArea
}

type ospfArea struct {
	LsaNumber        uint32
	LsaNetworkNumber uint32
	LsaSummaryNumber uint32
	LsaAsbrNumber    uint32
	LsaNssaNumber    uint32
}

func processOSPFNeigh(ch chan<- prometheus.Metric, jsonOSPFNeigh []byte, ospfDesc map[string]*prometheus.Desc, instanceID int) error {
	var vrfNeighs map[string]vrfNeighbors
	if err := json.Unmarshal(jsonOSPFNeigh, &vrfNeighs); err != nil {
		return fmt.Errorf("cannot unmarshal ospf neighbor json: %w", err)
	}
	for vrfName, vrfData := range vrfNeighs {
		for neighborName, neighbors := range vrfData.Neighbors {
			ospfNeighMetrics(ch, neighborName, neighbors, vrfName, ospfDesc, instanceID)
		}
	}

	return nil
}

func ospfNeighMetrics(ch chan<- prometheus.Metric, neighborName string, neighbors []ospfNeighbor, vrfName string, ospfDesc map[string]*prometheus.Desc, instanceID int) {
	var labels []string
	if instanceID != 0 {
		labels = append(labels, strconv.Itoa(instanceID))
	}
	labels = append(labels, strings.ToLower(vrfName), neighborName)
	for _, neighbor := range neighbors {
		var state float64
		switch neighbor.State {
		case "Down":
			state = 1
		case "Init":
			state = 2
		case "2-Way":
			state = 3
		case "ExStart":
			state = 4
		case "Exchange":
			state = 5
		case "Loading":
			state = 6
		case "Full":
			state = 7
		default:
			continue
		}
		newGauge(ch, ospfDesc["ospfNeighState"], state, append(labels, neighbor.IfaceName, neighbor.LocalAddress, neighbor.RemoteAddress)...)
	}
}

type vrfNeighbors struct {
	VRFName   string
	Neighbors map[string][]ospfNeighbor
}

func GetOSPFState(nbrState, state string) string {
	if nbrState != "" {
		return nbrState
	}
	return state
}

type ospfNeighbor struct {
	State         string `json:"state"`
	NbrState      string `json:"nbrState"`
	IfaceName     string `json:"ifaceName"`
	LocalAddress  string `json:"localAddress"`
	RemoteAddress string `json:"address"`
}

func (n *ospfNeighbor) UnmarshalJSON(data []byte) error {
	var temp struct {
		NbrState   string `json:"nbrState"`
		State      string `json:"state"`
		IfaceName  string `json:"ifaceName"`
		LocalAddr  string `json:"localAddress"`
		RemoteAddr string `json:"address"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("cannot unmarshal ospf neighbor json: %w", err)
	}

	iface := strings.Split(temp.IfaceName, ":")
	if len(iface) == 2 {
		n.IfaceName = iface[0]
		n.LocalAddress = iface[1]
	} else {
		return fmt.Errorf("cannot unmarshal ospf neighbor iface: %s", iface)
	}

	state := strings.Split(GetOSPFState(temp.NbrState, temp.State), "/")
	if len(state) > 0 {
		n.State = state[0]
	} else {
		return fmt.Errorf("cannot unmarshal ospf neighbor state: %s", state)
	}

	n.RemoteAddress = temp.RemoteAddr

	return nil
}

func processOSPFDataMaxAge(ch chan<- prometheus.Metric, jsonOSPFMaxAge []byte, ospfDesc map[string]*prometheus.Desc, instanceID int) error {
	var all map[string]ospfDataMaxAge
	if err := json.Unmarshal(jsonOSPFMaxAge, &all); err != nil {
		return fmt.Errorf("cannot unmarshal ospf max age json: %w", err)
	}

	for vrfName, vrfData := range all {
		ospfDataMaxAgeMetrics(ch, vrfData, vrfName, ospfDesc, instanceID)
	}
	return nil
}

func ospfDataMaxAgeMetrics(ch chan<- prometheus.Metric, ospfData ospfDataMaxAge, vrfName string, ospfDesc map[string]*prometheus.Desc, instanceID int) {
	labels := []string{strings.ToLower(vrfName)}
	if instanceID != 0 {
		labels = append(labels, strconv.Itoa(instanceID))
	}
	newGauge(ch, ospfDesc["ospfDataMaxAge"], float64(len(ospfData.MaxAgeLinkStates)), labels...)
}

type ospfDataMaxAge struct {
	VRFName          string
	MaxAgeLinkStates map[string]struct{}
}
