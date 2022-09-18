package collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	bgpSubsystem = "bgp"

	bgpPeerTypes          = kingpin.Flag("collector.bgp.peer-types", "Enable the frr_bgp_peer_types_up metric (default: disabled).").Default("False").Bool()
	frrBGPDescKey         = kingpin.Flag("collector.bgp.peer-types.keys", "Select the keys from the JSON formatted BGP peer description of which the values will be used with the frr_bgp_peer_types_up metric. Supports multiple values (default: type).").Default("type").Strings()
	bgpPeerDescs          = kingpin.Flag("collector.bgp.peer-descriptions", "Add the value of the desc key from the JSON formatted BGP peer description as a label to peer metrics. (default: disabled).").Default("False").Bool()
	bgpPeerDescsText      = kingpin.Flag("collector.bgp.peer-descriptions.plain-text", "Use the full text field of the BGP peer description instead of the value of the JSON formatted desc key (default: disabled).").Default("False").Bool()
	bgpAdvertisedPrefixes = kingpin.Flag("collector.bgp.advertised-prefixes", "Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer. This is an option for older versions of FRR that don't have PfxSent field (default: disabled).").Default("False").Bool()
)

func init() {
	registerCollector(bgpSubsystem, enabledByDefault, NewBGPCollector)
	registerCollector(bgpSubsystem+"6", disabledByDefault, NewBGP6Collector)
	registerCollector(bgpSubsystem+"l2vpn", disabledByDefault, NewBGPL2VPNCollector)
}

type bgpCollector struct {
	logger       log.Logger
	descriptions map[string]*prometheus.Desc
	afi          string
}

// NewBGPCollector collects BGP metrics, implemented as per the Collector interface.
func NewBGPCollector(logger log.Logger) (Collector, error) {
	return &bgpCollector{logger: logger, descriptions: getBGPDesc(), afi: "ipv4"}, nil
}

func getBGPDesc() map[string]*prometheus.Desc {

	bgpLabels := []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerTypeLabels := []string{"type", "afi", "safi"}
	bgpPeerLabels := append(bgpLabels, "peer", "peer_as")

	if *bgpPeerDescs {
		bgpPeerLabels = append(bgpLabels, "peer", "peer_as", "peer_desc")
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
func NewBGP6Collector(logger log.Logger) (Collector, error) {
	return &bgpCollector{logger: logger, descriptions: getBGPDesc(), afi: "ipv6"}, nil
}

type bgpL2VPNCollector struct {
	logger       log.Logger
	descriptions map[string]*prometheus.Desc
}

// NewBGPL2VPNCollector collects BGP L2VPN metrics, implemented as per the Collector interface.
func NewBGPL2VPNCollector(logger log.Logger) (Collector, error) {
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

func collectBGP(ch chan<- prometheus.Metric, AFI string, logger log.Logger, desc map[string]*prometheus.Desc) error {
	SAFI := ""

	if (AFI == "ipv4") || (AFI == "ipv6") {
		SAFI = "unicast"

	} else if AFI == "l2vpn" {
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

func processBGPSummary(ch chan<- prometheus.Metric, jsonBGPSum []byte, AFI string, SAFI string, logger log.Logger, bgpDesc map[string]*prometheus.Desc) error {
	var jsonMap map[string]bgpProcess
	if err := json.Unmarshal(jsonBGPSum, &jsonMap); err != nil {
		return err
	}

	var peerDescJSON map[string]map[string]map[string]string
	var peerDescText map[string]map[string]string
	var err error
	if *bgpPeerTypes || *bgpPeerDescs {
		peerDescJSON, peerDescText, err = getBGPPeerDesc(logger)
		if err != nil {
			return err
		}
	}

	peerTypes := make(map[string]float64)
	wgAdvertisedPrefixes := &sync.WaitGroup{}
	for vrfName, vrfData := range jsonMap {
		// The labels are "vrf", "afi",  "safi", "local_as"
		localAs := strconv.FormatUint(uint64(vrfData.AS), 10)
		procLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs}
		// No point collecting metrics if no peers configured.
		if vrfData.PeerCount != 0 {
			newGauge(ch, bgpDesc["ribCount"], float64(vrfData.RIBCount), procLabels...)
			newGauge(ch, bgpDesc["ribMemory"], float64(vrfData.RIBMemory), procLabels...)
			newGauge(ch, bgpDesc["peerCount"], float64(vrfData.PeerCount), procLabels...)
			newGauge(ch, bgpDesc["peerMemory"], float64(vrfData.PeerMemory), procLabels...)
			newGauge(ch, bgpDesc["peerGroupCount"], float64(vrfData.PeerGroupCount), procLabels...)
			newGauge(ch, bgpDesc["peerGroupMemory"], float64(vrfData.PeerGroupMemory), procLabels...)

			for peerIP, peerData := range vrfData.Peers {
				// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as"
				peerLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs, peerIP, strconv.FormatUint(uint64(peerData.RemoteAs), 10)}

				if *bgpPeerDescs {
					d := ""
					if *bgpPeerDescsText {
						d = peerDescText[vrfName][peerIP]
					} else {
						d = peerDescJSON[vrfName][peerIP]["desc"]
					}
					// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as", "peer_desc"
					peerLabels = append(peerLabels, d)
				}

				// In earlier versions of FRR did not expose a summary of advertised prefixes for all peers, but in later versions it can get with PfxSnt field.
				if peerData.PfxSnt != nil {
					newGauge(ch, bgpDesc["prefixAdvertisedCount"], float64(*peerData.PfxSnt), peerLabels...)
				} else if *bgpAdvertisedPrefixes {
					wgAdvertisedPrefixes.Add(1)
					go getPeerAdvertisedPrefixes(ch, wgAdvertisedPrefixes, AFI, SAFI, vrfName, peerIP, logger, bgpDesc, peerLabels...)
				}

				newCounter(ch, bgpDesc["msgRcvd"], float64(peerData.MsgRcvd), peerLabels...)
				newCounter(ch, bgpDesc["msgSent"], float64(peerData.MsgSent), peerLabels...)
				newGauge(ch, bgpDesc["UptimeSec"], float64(peerData.PeerUptimeMsec)*0.001, peerLabels...)

				// In earlier versions of FRR, the prefixReceivedCount JSON element is used for the number of recieved prefixes, but in later versions it was changed to PfxRcd.
				prefixReceived := 0.0
				if peerData.PrefixReceivedCount != 0 {
					prefixReceived = float64(peerData.PrefixReceivedCount)
				} else if peerData.PfxRcd != 0 {
					prefixReceived = float64(peerData.PfxRcd)
				}
				newGauge(ch, bgpDesc["prefixReceivedCount"], prefixReceived, peerLabels...)

				if *bgpPeerTypes {
					for _, descKey := range *frrBGPDescKey {
						if peerDescJSON[vrfName][peerIP][descKey] != "" {
							if _, exist := peerTypes[strings.TrimSpace(peerDescJSON[vrfName][peerIP][descKey])]; !exist {
								peerTypes[strings.TrimSpace(peerDescJSON[vrfName][peerIP][descKey])] = 0
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
							if peerDescJSON[vrfName][peerIP][descKey] != "" {
								peerTypes[strings.TrimSpace(peerDescJSON[vrfName][peerIP][descKey])]++
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

	wgAdvertisedPrefixes.Wait()

	for peerType, count := range peerTypes {
		peerTypeLabels := []string{peerType, strings.ToLower(AFI), strings.ToLower(SAFI)}
		newGauge(ch, bgpDesc["peerTypesUp"], count, peerTypeLabels...)
	}
	return nil
}

func getPeerAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, AFI string, SAFI string, vrfName string, neighbor string, logger log.Logger, bgpDesc map[string]*prometheus.Desc, peerLabels ...string) {
	defer wg.Done()

	var cmd string
	if strings.ToLower(vrfName) == "default" {
		cmd = fmt.Sprintf("show bgp  %s %s neighbors %s advertised-routes json", AFI, SAFI, neighbor)
	} else {
		cmd = fmt.Sprintf("show bgp vrf %s %s %s neighbors %s advertised-routes json", vrfName, AFI, SAFI, neighbor)
	}

	output, err := executeBGPCommand(cmd)
	if err != nil {
		level.Error(logger).Log("msg", "get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	var advertisedPrefixes bgpAdvertisedRoutes
	if err := json.Unmarshal(output, &advertisedPrefixes); err != nil {
		level.Error(logger).Log("msg", "get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	newGauge(ch, bgpDesc["prefixAdvertisedCount"], float64(advertisedPrefixes.TotalPrefixCounter), peerLabels...)
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
}
type bgpAdvertisedRoutes struct {
	TotalPrefixCounter uint32 `json:"totalPrefixCounter"`
}

// Returns:
//  - Map from JSON formatted BGP peer descriptions
//  - Plain text description of peers
//  - Error
func getBGPPeerDesc(logger log.Logger) (map[string]map[string]map[string]string, map[string]map[string]string, error) {
	cmd := "show bgp vrf all neighbors json"
	output, err := executeBGPCommand("show bgp vrf all neighbors json")
	if err != nil {
		return nil, nil, err
	}
	return processBGPPeerDesc(logger, output, cmd)
}

func processBGPPeerDesc(logger log.Logger, output []byte, cmd string) (map[string]map[string]map[string]string, map[string]map[string]string, error) {

	// Expected map format: map["vrf"]["peer IP"]["json desc field"]["json desc value"]
	descJSON := make(map[string]map[string]map[string]string)

	// Expected map format: map["vrf"]["peer IP"]["text desc"]
	descText := make(map[string]map[string]string)

	// Unfortunately, the 'show bgp vrf all neighbors json' output is poorly structured -- neighbors are
	// fields on the same level of the vrfName and vrfId field. As such, loop through each key and apply
	// logic to determine whether the key is a neighbor.
	//
	// Example:
	// {
	//    "default":{
	//      "vrfId":-1,
	//      "vrfName":"default",
	//      "swp2":{
	// 	      "nbrDesc":"desc"
	//       },
	//      "10.189.0.178":{
	// 	      "nbrDesc":"desc"
	//      }
	//    },
	//    "vrf1":{
	//      "vrfId":-1,
	//      "vrfName":"vrf1",
	//      "swp1":{
	// 	      "nbrDesc":"desc"
	//      }
	//    }
	// }
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(output, &jsonMap); err != nil {
		level.Error(logger).Log("msg", "cannot unmarshal bgp neighbors", "command", cmd, "command output", string(output), "err", err)
		return nil, nil, err
	}

	for vrfName, vrfData := range jsonMap {
		var vrfFields map[string]json.RawMessage

		if err := json.Unmarshal(vrfData, &vrfFields); err != nil {
			level.Error(logger).Log("msg", "cannot unmarshal bgp neighbors vrf data", "data", string(vrfData), "command", cmd, "command output", string(output), "err", err)
			return nil, nil, err
		}

		for neighbor, neighborValues := range vrfFields {
			switch neighbor {
			case "vrfName", "vrfId":
				// Do nothing as we do not need the value of these fields.
			default:
				// All other fields are neighbors.
				var neighborData bgpBGPNeighbor
				if err := json.Unmarshal(neighborValues, &neighborData); err != nil {
					level.Error(logger).Log("msg", "cannot unmarshal bgp neighbors bgp neighbor data", "data", string(neighborValues), "command", cmd, "command output", string(output), "err", err)
					return nil, nil, err
				}

				if !*bgpPeerDescsText {

					var peerDesc map[string]string
					if err := json.Unmarshal([]byte(neighborData.NbrDesc), &peerDesc); err != nil {
						// Don't return an error as unmarshalling is best effort.
						level.Error(logger).Log("msg", "cannot unmarshall bgp description", "description", neighborData.NbrDesc, "neighbor", neighbor, "err", err)
					}
					if _, exists := descJSON[vrfName]; !exists {
						descJSON[vrfName] = make(map[string]map[string]string)
					}
					descJSON[vrfName][neighbor] = peerDesc
				}
				if _, exists := descText[vrfName]; !exists {
					descText[vrfName] = make(map[string]string)
				}
				descText[vrfName][neighbor] = neighborData.NbrDesc
			}
		}
	}
	return descJSON, descText, nil
}

type bgpBGPNeighbor struct {
	NbrDesc string `json:"nbrDesc"`
}
