package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	bgpSubsystem = "bgp"

	bgpPeerTypes                = kingpin.Flag("collector.bgp.peer-types", "Enable the frr_bgp_peer_types_up metric (default: disabled).").Default("False").Bool()
	frrBGPDescKey               = kingpin.Flag("collector.bgp.peer-types.keys", "Select the keys from the JSON formatted BGP peer description of which the values will be used with the frr_bgp_peer_types_up metric. Supports multiple values (default: type).").Default("type").Strings()
	bgpPeerDescs                = kingpin.Flag("collector.bgp.peer-descriptions", "Add the value of the desc key from the JSON formatted BGP peer description as a label to peer metrics. (default: disabled).").Default("False").Bool()
	bgpPeerGroups               = kingpin.Flag("collector.bgp.peer-groups", "Adds the peer's peer group name as a label. (default: disabled).").Default("False").Bool()
	bgpPeerHostnames            = kingpin.Flag("collector.bgp.peer-hostnames", "Adds the peer's hostname as a label. (default: disabled).").Default("False").Bool()
	bgpPeerDescsText            = kingpin.Flag("collector.bgp.peer-descriptions.plain-text", "Use the full text field of the BGP peer description instead of the value of the JSON formatted desc key (default: disabled).").Default("False").Bool()
	bgpAdvertisedPrefixes       = kingpin.Flag("collector.bgp.advertised-prefixes", "Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer. This is an option for older versions of FRR that don't have PfxSent field (default: disabled).").Default("False").Bool()
	bgpAcceptedFilteredPrefixes = kingpin.Flag("collector.bgp.accepted-filtered-prefixes", "Enable retrieval of accepted and filtered BGP prefix counts (default: disabled).").Default("False").Bool()
	bgpNextHopInterface         = kingpin.Flag("collector.bgp.next-hop-interface", "Add next-hop interface to bgp peer state metric (default: disabled).").Default("False").Bool()
)

func init() {
	registerCollector(bgpSubsystem, enabledByDefault, NewBGPCollector)
	registerCollector(bgpSubsystem+"6", disabledByDefault, NewBGP6Collector)
	registerCollector(bgpSubsystem+"l2vpn", disabledByDefault, NewBGPL2VPNCollector)
}

type bgpCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
	afi          string
}

// NewBGPCollector collects BGP metrics, implemented as per the Collector interface.
func NewBGPCollector(logger *slog.Logger) (Collector, error) {
	return &bgpCollector{logger: logger, descriptions: getBGPDesc(), afi: "ipv4"}, nil
}

func getBGPDesc() map[string]*prometheus.Desc {
	bgpLabels := []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerTypeLabels := []string{"type", "afi", "safi"}
	bgpPeerLabels := append(bgpLabels, "peer", "peer_as")

	if *bgpPeerDescs {
		bgpPeerLabels = append(bgpPeerLabels, "peer_desc")
	}

	if *bgpPeerHostnames {
		bgpPeerLabels = append(bgpPeerLabels, "peer_hostname")
	}

	if *bgpPeerGroups {
		bgpPeerLabels = append(bgpPeerLabels, "peer_group")
	}

	if *bgpNextHopInterface {
		bgpPeerLabels = append(bgpPeerLabels, "nexthop_interface")
	}

	return map[string]*prometheus.Desc{
		"ribCount":              colPromDesc(bgpSubsystem, "rib_count_total", "Number of routes in the RIB.", bgpLabels),
		"ribMemory":             colPromDesc(bgpSubsystem, "rib_memory_bytes", "Memory consumbed by the RIB.", bgpLabels),
		"peerCount":             colPromDesc(bgpSubsystem, "peers_count_total", "Number peers configured.", bgpLabels),
		"peerMemory":            colPromDesc(bgpSubsystem, "peers_memory_bytes", "Memory consumed by peers.", bgpLabels),
		"peerGroupCount":        colPromDesc(bgpSubsystem, "peer_groups_count_total", "Number of peer groups configured.", bgpLabels),
		"peerGroupMemory":       colPromDesc(bgpSubsystem, "peer_groups_memory_bytes", "Memory consumed by peer groups.", bgpLabels),
		"msgRcvd":               colPromDesc(bgpSubsystem, "peer_message_received_total", "Number of received messages.", bgpPeerLabels),
		"msgSent":               colPromDesc(bgpSubsystem, "peer_message_sent_total", "Number of sent messages.", bgpPeerLabels),
		"prefixReceivedCount":   colPromDesc(bgpSubsystem, "peer_prefixes_received_count_total", "Number of prefixes received.", bgpPeerLabels),
		"prefixAdvertisedCount": colPromDesc(bgpSubsystem, "peer_prefixes_advertised_count_total", "Number of prefixes advertised.", bgpPeerLabels),
		"prefixAcceptedCount":   colPromDesc(bgpSubsystem, "peer_prefixes_accepted_count_total", "Number of prefixes accepted.", bgpPeerLabels),
		"prefixFilteredCount":   colPromDesc(bgpSubsystem, "peer_prefixes_filtered_count_total", "Number of prefixes filtered.", bgpPeerLabels),
		"state":                 colPromDesc(bgpSubsystem, "peer_state", "State of the peer (2 = Administratively Down, 1 = Established, 0 = Down).", bgpPeerLabels),
		"UptimeSec":             colPromDesc(bgpSubsystem, "peer_uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":           colPromDesc(bgpSubsystem, "peer_types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}
}

// Update implemented as per the Collector interface.
func (c *bgpCollector) Update(ch chan<- prometheus.Metric) error {
	return collectBGP(ch, c.afi, c.logger, c.descriptions)
}

// NewBGP6Collector collects BGPv6 metrics, implemented as per the Collector interface.
func NewBGP6Collector(logger *slog.Logger) (Collector, error) {
	return &bgpCollector{logger: logger, descriptions: getBGPDesc(), afi: "ipv6"}, nil
}

type bgpL2VPNCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewBGPL2VPNCollector collects BGP L2VPN metrics, implemented as per the Collector interface.
func NewBGPL2VPNCollector(logger *slog.Logger) (Collector, error) {
	return &bgpL2VPNCollector{logger: logger, descriptions: getBGPL2VPNDesc()}, nil
}

func getBGPL2VPNDesc() map[string]*prometheus.Desc {
	bgpDesc := getBGPDesc()
	labels := []string{"vni", "type", "vxlanIf", "tenantVrf"}
	metricPrefix := "bgp_l2vpn_evpn"

	bgpDesc["numMacs"] = colPromDesc(metricPrefix, "mac_count_total", "Number of known MAC addresses", labels)
	bgpDesc["numArpNd"] = colPromDesc(metricPrefix, "arp_nd_count_total", "Number of ARP / ND entries", labels)
	bgpDesc["numRemoteVteps"] = colPromDesc(metricPrefix, "remote_vtep_count_total", "Number of known remote VTEPs. A value of -1 indicates a non-integer output from FRR, such as n/a.", labels)

	return bgpDesc
}

// Update implemented as per the Collector interface.
func (c *bgpL2VPNCollector) Update(ch chan<- prometheus.Metric) error {
	if err := collectBGP(ch, "l2vpn", c.logger, c.descriptions); err != nil {
		return err
	}
	cmd := "show evpn vni json"
	jsonBGPL2vpnEvpnSum, err := executeZebraCommand(cmd)
	if err != nil {
		return err
	}
	if len(jsonBGPL2vpnEvpnSum) == 0 {
		return nil
	}
	if err := processBgpL2vpnEvpnSummary(ch, jsonBGPL2vpnEvpnSum, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(jsonBGPL2vpnEvpnSum), err)
	}
	return nil
}

type vxLanStats struct {
	Vni            uint32
	VxlanType      string `json:"type"`
	VxlanIf        string
	NumMacs        uint32
	NumArpNd       uint32
	NumRemoteVteps interface{} // it's possible for the numRemoteVteps field to contain non-int values such as "n\/a"
	TenantVrf      string
}

func processBgpL2vpnEvpnSummary(ch chan<- prometheus.Metric, jsonBGPL2vpnEvpnSum []byte, bgpL2vpnDesc map[string]*prometheus.Desc) error {
	var jsonMap map[string]vxLanStats
	if err := json.Unmarshal(jsonBGPL2vpnEvpnSum, &jsonMap); err != nil {
		return err
	}

	for _, vxLanStat := range jsonMap {
		bgpL2vpnLabels := []string{strconv.FormatUint(uint64(vxLanStat.Vni), 10), vxLanStat.VxlanType, vxLanStat.VxlanIf, vxLanStat.TenantVrf}
		newGauge(ch, bgpL2vpnDesc["numMacs"], float64(vxLanStat.NumMacs), bgpL2vpnLabels...)
		newGauge(ch, bgpL2vpnDesc["numArpNd"], float64(vxLanStat.NumArpNd), bgpL2vpnLabels...)
		remoteVteps, ok := vxLanStat.NumRemoteVteps.(float64)
		if !ok {
			remoteVteps = -1
		}
		newGauge(ch, bgpL2vpnDesc["numRemoteVteps"], remoteVteps, bgpL2vpnLabels...)

	}
	return nil
}

func collectBGP(ch chan<- prometheus.Metric, AFI string, logger *slog.Logger, desc map[string]*prometheus.Desc) error {
	SAFI := ""

	switch AFI {
	case "ipv4", "ipv6":
		SAFI = ""
	case "l2vpn":
		SAFI = "evpn"
	}
	cmd := fmt.Sprintf("show bgp vrf all %s %s summary json", AFI, SAFI)
	jsonBGPSum, err := executeBGPCommand(cmd)
	if err != nil {
		return err
	}
	if err := processBGPSummary(ch, jsonBGPSum, AFI, SAFI, logger, desc); err != nil {
		return cmdOutputProcessError(cmd, string(jsonBGPSum), err)
	}
	return nil
}

func processBGPSummary(ch chan<- prometheus.Metric, jsonBGPSum []byte, AFI string, SAFI string, logger *slog.Logger, bgpDesc map[string]*prometheus.Desc) error {
	var jsonMap map[string]map[string]bgpProcess

	// if we've specified SAFI in the command, we won't have the SAFI layer of array to loop through
	// so we simulate it here, rather than using a conditional and writing almost the same code twice
	if AFI == "l2vpn" && SAFI == "evpn" {
		// since we need to massage the format a bit, unmarshall into a temp variable
		var tempJSONMap map[string]bgpProcess
		if err := json.Unmarshal(jsonBGPSum, &tempJSONMap); err != nil {
			return err
		}
		jsonMap = map[string]map[string]bgpProcess{}
		for vrfName, vrfData := range tempJSONMap {
			jsonMap[vrfName] = map[string]bgpProcess{"xxxxevpn": vrfData}
		}
	} else {
		// we have the format we expect, unmarshall directly into jsonMap
		if err := json.Unmarshal(jsonBGPSum, &jsonMap); err != nil {
			return err
		}
	}

	var peerDesc map[string]bgpVRF
	var err error
	if *bgpPeerTypes || *bgpPeerDescs || *bgpPeerGroups {
		peerDesc, err = getBGPPeerDesc()
		if err != nil {
			return err
		}
	}

	var bgpNextHop map[string]bgpNextHop
	if *bgpNextHopInterface {
		bgpNextHop, err = getBGPNexthop()
		if err != nil {
			return err
		}
	}

	peerTypes := make(map[string]map[string]float64)
	wg := &sync.WaitGroup{}
	for vrfName, vrfData := range jsonMap {
		for safiName, safiData := range vrfData {
			// The labels are "vrf", "afi",  "safi", "local_as"
			localAs := strconv.FormatUint(uint64(safiData.AS), 10)
			procLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(safiName[4:]), localAs}
			// No point collecting metrics if no peers configured.
			if safiData.PeerCount != 0 {
				newGauge(ch, bgpDesc["ribCount"], float64(safiData.RIBCount), procLabels...)
				newGauge(ch, bgpDesc["ribMemory"], float64(safiData.RIBMemory), procLabels...)
				newGauge(ch, bgpDesc["peerCount"], float64(safiData.PeerCount), procLabels...)
				newGauge(ch, bgpDesc["peerMemory"], float64(safiData.PeerMemory), procLabels...)
				newGauge(ch, bgpDesc["peerGroupCount"], float64(safiData.PeerGroupCount), procLabels...)
				newGauge(ch, bgpDesc["peerGroupMemory"], float64(safiData.PeerGroupMemory), procLabels...)

				for peerIP, peerData := range safiData.Peers {
					// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as"
					peerLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(safiName[4:]), localAs, peerIP, strconv.FormatUint(uint64(peerData.RemoteAs), 10)}

					if *bgpPeerDescs {
						d := peerDesc[vrfName].BGPNeighbors[peerIP].Desc
						if *bgpPeerDescsText {
							// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as", "peer_desc"
							peerLabels = append(peerLabels, d)
						} else {
							// Assume the FRR BGP neighbor description is JSON formatted, and the description is in the "desc" field.
							jsonDesc := struct{ Desc string }{}
							if err := json.Unmarshal([]byte(d), &jsonDesc); err != nil {
								// Don't return an error as unmarshalling is best effort.
								logger.Error("cannot unmarshal bgp description", "description", peerDesc[vrfName].BGPNeighbors[peerIP].Desc, "err", err)
							}
							// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as", "peer_desc"
							peerLabels = append(peerLabels, jsonDesc.Desc)
						}
					}

					if *bgpPeerHostnames {
						peerLabels = append(peerLabels, peerData.Hostname)
					}

					if *bgpPeerGroups {
						peerLabels = append(peerLabels, peerDesc[vrfName].BGPNeighbors[peerIP].PeerGroup)
					}

					if *bgpNextHopInterface {
						var familyMap = map[string]map[string]bgpNextHopInterfaces{
							"ipv4": bgpNextHop[vrfName].IPv4,
							"ipv6": bgpNextHop[vrfName].IPv6,
						}
						key := strings.ToLower(AFI)
						if nexthop, ok := familyMap[key][peerIP]; !ok {
							logger.Warn("BGP next hop not found", "afi", AFI, "peer", peerIP)
							peerLabels = append(peerLabels, "unknown")
						} else {
							if len(nexthop.Nexthops) > 0 {
								peerLabels = append(peerLabels, nexthop.Nexthops[0].InterfaceName)
							} else {
								peerLabels = append(peerLabels, "unknown")
							}
						}
					}

					// In earlier versions of FRR did not expose a summary of advertised prefixes for all peers, but in later versions it can get with PfxSnt field.
					if peerData.PfxSnt != nil {
						newGauge(ch, bgpDesc["prefixAdvertisedCount"], float64(*peerData.PfxSnt), peerLabels...)
					} else if *bgpAdvertisedPrefixes {
						wg.Add(1)
						go getPeerAdvertisedPrefixes(ch, wg, AFI, safiName[4:], vrfName, peerIP, logger, bgpDesc, peerLabels...)
					}

					newCounter(ch, bgpDesc["msgRcvd"], float64(peerData.MsgRcvd), peerLabels...)
					newCounter(ch, bgpDesc["msgSent"], float64(peerData.MsgSent), peerLabels...)
					newGauge(ch, bgpDesc["UptimeSec"], float64(peerData.PeerUptimeMsec)*0.001, peerLabels...)

					// In earlier versions of FRR, the prefixReceivedCount JSON element is used for the number of received prefixes, but in later versions it was changed to PfxRcd.
					prefixReceived := 0.0
					if peerData.PrefixReceivedCount != 0 {
						prefixReceived = float64(peerData.PrefixReceivedCount)
					} else if peerData.PfxRcd != 0 {
						prefixReceived = float64(peerData.PfxRcd)
					}
					newGauge(ch, bgpDesc["prefixReceivedCount"], prefixReceived, peerLabels...)

					if *bgpAcceptedFilteredPrefixes {
						wg.Add(1)
						go getPeerAcceptedFilteredRoutes(ch, wg, AFI, safiName[4:], vrfName, peerIP, prefixReceived, logger, bgpDesc, peerLabels...)
					}

					var peerDescTypes map[string]string
					if *bgpPeerTypes {
						if err := json.Unmarshal([]byte(peerDesc[vrfName].BGPNeighbors[peerIP].Desc), &peerDescTypes); err != nil {
							// Don't return an error as unmarshalling is best effort.
							logger.Error("cannot unmarshal bgp description", "description", peerDesc[vrfName].BGPNeighbors[peerIP].Desc, "err", err)
						}

						// add key for this SAFI if it doesn't exist
						if _, exist := peerTypes[strings.ToLower(safiName[4:])]; !exist {
							peerTypes[strings.ToLower(safiName[4:])] = make(map[string]float64)
						}

						for _, descKey := range *frrBGPDescKey {
							if peerDescTypes[descKey] != "" {
								if _, exist := peerTypes[strings.ToLower(safiName[4:])][strings.TrimSpace(peerDescTypes[descKey])]; !exist {
									peerTypes[strings.ToLower(safiName[4:])][strings.TrimSpace(peerDescTypes[descKey])] = 0
								}
							}
						}
					}
					peerState := 0.0
					switch peerDataState := strings.ToLower(peerData.State); peerDataState {
					case "established":
						peerState = 1
						if *bgpPeerTypes {
							for _, descKey := range *frrBGPDescKey {
								if peerDescTypes[descKey] != "" {
									peerTypes[strings.ToLower(safiName[4:])][strings.TrimSpace(peerDescTypes[descKey])]++
								}
							}
						}
					case "idle (admin)":
						peerState = 2
					}
					newGauge(ch, bgpDesc["state"], peerState, peerLabels...)

				}
			}
		}
	}

	wg.Wait()

	for peerSafi, peerTypesPerSafi := range peerTypes {
		for peerType, count := range peerTypesPerSafi {
			peerTypeLabels := []string{peerType, strings.ToLower(AFI), peerSafi}
			newGauge(ch, bgpDesc["peerTypesUp"], count, peerTypeLabels...)
		}
	}
	return nil
}

func getPeerAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, AFI string, SAFI string, vrfName string, neighbor string, logger *slog.Logger, bgpDesc map[string]*prometheus.Desc, peerLabels ...string) {
	defer wg.Done()

	var cmd string
	if strings.ToLower(vrfName) == "default" {
		cmd = fmt.Sprintf("show bgp  %s %s neighbors %s advertised-routes json", AFI, SAFI, neighbor)
	} else {
		cmd = fmt.Sprintf("show bgp vrf %s %s %s neighbors %s advertised-routes json", vrfName, AFI, SAFI, neighbor)
	}

	output, err := executeBGPCommand(cmd)
	if err != nil {
		logger.Error("get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	var advertisedPrefixes bgpAdvertisedRoutes
	if err := json.Unmarshal(output, &advertisedPrefixes); err != nil {
		logger.Error("get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	newGauge(ch, bgpDesc["prefixAdvertisedCount"], float64(advertisedPrefixes.TotalPrefixCounter), peerLabels...)
}

type bgpRoutes struct {
	// We care only about the routes
	Routes map[string][]json.RawMessage `json:"routes"`
}

func getPeerAcceptedFilteredRoutes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, AFI string, SAFI string, vrfName string, neighbor string, prefixesReceived float64, logger *slog.Logger, bgpDesc map[string]*prometheus.Desc, peerLabels ...string) {
	defer wg.Done()

	var cmd string
	if strings.ToLower(vrfName) == "default" {
		cmd = fmt.Sprintf("show bgp  %s %s neighbors %s routes json", strings.ToLower(AFI), strings.ToLower(SAFI), neighbor)
	} else {
		cmd = fmt.Sprintf("show bgp vrf %s %s %s neighbors %s routes json", vrfName, strings.ToLower(AFI), strings.ToLower(SAFI), neighbor)
	}

	output, err := executeBGPCommand(cmd)
	if err != nil {
		logger.Error("get neighbor accepted filtered routes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	var routes bgpRoutes
	if err := json.Unmarshal(output, &routes); err != nil {
		logger.Error("get neighbor accepted filtered routes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	prefixesAccepted := float64(len(routes.Routes))

	newGauge(ch, bgpDesc["prefixAcceptedCount"], prefixesAccepted, peerLabels...)
	newGauge(ch, bgpDesc["prefixFilteredCount"], prefixesReceived-prefixesAccepted, peerLabels...)
}

type bgpProcess struct {
	RouterID        string
	AS              uint32
	RIBCount        uint32
	RIBMemory       uint32
	PeerCount       uint32
	PeerMemory      uint32
	PeerGroupCount  uint32
	PeerGroupMemory uint32
	Peers           map[string]*bgpPeerSession
}

type bgpPeerSession struct {
	State               string
	RemoteAs            uint32
	MsgRcvd             uint32
	MsgSent             uint32
	PeerUptimeMsec      uint64
	PrefixReceivedCount uint32
	PfxRcd              uint32
	PfxSnt              *uint32
	Hostname            string
}
type bgpAdvertisedRoutes struct {
	TotalPrefixCounter uint32 `json:"totalPrefixCounter"`
}

func getBGPPeerDesc() (map[string]bgpVRF, error) {
	output, err := executeBGPCommand("show bgp vrf all neighbors json")
	if err != nil {
		return nil, err
	}
	return processBGPPeerDesc(output)
}

func processBGPPeerDesc(output []byte) (map[string]bgpVRF, error) {
	vrfMap := make(map[string]bgpVRF)
	if err := json.Unmarshal([]byte(output), &vrfMap); err != nil {
		return nil, err
	}
	return vrfMap, nil
}

func (vrf *bgpVRF) UnmarshalJSON(data []byte) error {
	var raw map[string]*json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	vrf.BGPNeighbors = make(map[string]bgpNeighbor)

	for k, v := range raw {
		switch k {
		case "vrfId":
			if err := json.Unmarshal(*v, &vrf.ID); err != nil {
				return err
			}
		case "vrfName":
			// This is somewhat redundant, since the VRF name is a top-level key in the source JSON.
			if err := json.Unmarshal(*v, &vrf.Name); err != nil {
				return err
			}
		default:
			var neighbor bgpNeighbor
			if err := json.Unmarshal(*v, &neighbor); err != nil {
				return err
			}

			vrf.BGPNeighbors[k] = neighbor
		}
	}
	return nil
}

type bgpVRF struct {
	ID           int                    `json:"vrfId"`
	Name         string                 `json:"vrfName"`
	BGPNeighbors map[string]bgpNeighbor `json:"-"`
}

type bgpNeighbor struct {
	Desc      string `json:"nbrDesc"`
	PeerGroup string `json:"peerGroup"`
}

type bgpNextHopInterfaces struct {
	Nexthops []struct {
		InterfaceName string `json:"interfaceName"`
	} `json:"nexthops"`
}

type bgpNextHop struct {
	IPv4 map[string]bgpNextHopInterfaces
	IPv6 map[string]bgpNextHopInterfaces
}

func getBGPNexthop() (map[string]bgpNextHop, error) {
	output, err := executeBGPCommand("show ip bgp vrf all nexthop json")
	if err != nil {
		return nil, err
	}
	return processBGPNexthop(output)
}

func processBGPNexthop(output []byte) (map[string]bgpNextHop, error) {
	bgpNextHop := make(map[string]bgpNextHop)
	if err := json.Unmarshal([]byte(output), &bgpNextHop); err != nil {
		return nil, err
	}
	return bgpNextHop, nil
}
