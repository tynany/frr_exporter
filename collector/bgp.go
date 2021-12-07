package collector

import (
	"encoding/json"
	"fmt"
	"regexp"
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

	bgpDescMtx   = sync.Mutex{}
	bgpDesc      map[string]*prometheus.Desc
	bgpL2vpnDesc map[string]*prometheus.Desc

	bgpPeerTypes          = kingpin.Flag("collector.bgp.peer-types", "Enable the frr_bgp_peer_types_up metric (default: disabled).").Default("False").Bool()
	frrBGPDescKey         = kingpin.Flag("collector.bgp.peer-types.keys", "Select the keys from the JSON formatted BGP peer description of which the values will be used with the frr_bgp_peer_types_up metric. Supports multiple values (default: type).").Default("type").Strings()
	bgpPeerDescs          = kingpin.Flag("collector.bgp.peer-descriptions", "Add the value of the desc key from the JSON formatted BGP peer description as a label to peer metrics. (default: disabled).").Default("False").Bool()
	bgpPeerDescsText      = kingpin.Flag("collector.bgp.peer-descriptions.plain-text", "Use the full text field of the BGP peer description instead of the value of the JSON formatted desc key (default: disabled).").Default("False").Bool()
	bgpAdvertisedPrefixes = kingpin.Flag("collector.bgp.advertised-prefixes", "Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer. This is an option for older versions of FRR that don't have PfxSent field (default: disabled).").Default("False").Bool()
)

func init() {
	registerCollector(bgpSubsystem, enabledByDefault, NewBGPCollector)
	registerCollector(bgpSubsystem+"6", enabledByDefault, NewBGP6Collector)
	registerCollector(bgpSubsystem+"l2vpn", enabledByDefault, NewBGPL2VPNCollector)
}

type bgpCollector struct {
	logger log.Logger
}

// NewBGPCollector collects BGP metrics, implemented as per the Collector interface.
func NewBGPCollector(logger log.Logger) (Collector, error) {
	return &bgpCollector{logger: logger}, nil
}

// Update implemented as per the Collector interface.
func (c *bgpCollector) Update(ch chan<- prometheus.Metric) error {
	return collectBGP(ch, "ipv4", c.logger)
}

type bgp6Collector struct {
	logger log.Logger
}

// NewBGP6Collector collects BGPv6 metrics, implemented as per the Collector interface.
func NewBGP6Collector(logger log.Logger) (Collector, error) {
	return &bgp6Collector{
		logger: logger,
	}, nil
}

// Update implemented as per the Collector interface.
func (c *bgp6Collector) Update(ch chan<- prometheus.Metric) error {
	return collectBGP(ch, "ipv6", c.logger)
}

type bgpL2VPNCollector struct {
	logger log.Logger
}

// NewBGPL2VPNCollector collects BGP L2VPN metrics, implemented as per the Collector interface.
func NewBGPL2VPNCollector(logger log.Logger) (Collector, error) {
	return &bgpL2VPNCollector{
		logger: logger,
	}, nil
}

// Update implemented as per the Collector interface.
func (c *bgpL2VPNCollector) Update(ch chan<- prometheus.Metric) error {
	if err := collectBGP(ch, "l2vpn", c.logger); err != nil {
		return err
	}

	jsonBGPL2vpnEvpnSum, err := getBgpL2vpnEvpnSummary()
	if err != nil {
		return fmt.Errorf("cannot execute 'show evpn vni json': %s", err)
	} else if len(jsonBGPL2vpnEvpnSum) != 0 {
		if err := processBgpL2vpnEvpnSummary(ch, jsonBGPL2vpnEvpnSum); err != nil {
			return err
		}
	}
	return nil
}

func getBgpL2vpnEvpnSummary() ([]byte, error) {
	return execVtyshCommand("-c", "show evpn vni json")
}

type vxLanStats struct {
	Vni            int
	VxlanType      string `json:"type"`
	VxlanIf        string
	NumMacs        float64
	NumArpNd       float64
	NumRemoteVteps interface{} // it's possible for the numRemoteVteps field to contain non-int values such as "n\/a"
	TenantVrf      string
}

func processBgpL2vpnEvpnSummary(ch chan<- prometheus.Metric, jsonBGPL2vpnEvpnSum []byte) error {
	var jsonMap map[string]vxLanStats
	if err := json.Unmarshal(jsonBGPL2vpnEvpnSum, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal outputs of 'show evpn vni json': %s", err)
	}

	for _, vxLanStat := range jsonMap {
		bgpL2vpnLabels := []string{strconv.Itoa(vxLanStat.Vni), vxLanStat.VxlanType, vxLanStat.VxlanIf, vxLanStat.TenantVrf}
		newGauge(ch, bgpL2vpnDesc["numMacs"], vxLanStat.NumMacs, bgpL2vpnLabels...)
		newGauge(ch, bgpL2vpnDesc["numArpNd"], vxLanStat.NumArpNd, bgpL2vpnLabels...)
		remoteVteps, ok := vxLanStat.NumRemoteVteps.(float64)
		if !ok {
			remoteVteps = -1
		}
		newGauge(ch, bgpL2vpnDesc["numRemoteVteps"], remoteVteps, bgpL2vpnLabels...)

	}
	return nil
}

// Helper function to set Prometheus metrics descriptions. BGPv4 metrics can have optional
// labels (i.e. peer, peer_as, peer_desc) as defined via the --collector.bgp.peer-descriptions flag,
// so it's not possible to set the descriptions via a global var.
func setBgpDesc() {
	if bgpDesc != nil {
		return
	}
	bgpDescMtx.Lock()
	defer bgpDescMtx.Unlock()
	bgpLabels := []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerTypeLabels := []string{"type", "afi", "safi"}
	bgpPeerLabels := append(bgpLabels, "peer", "peer_as")

	if *bgpPeerDescs {
		bgpPeerLabels = append(bgpLabels, "peer", "peer_as", "peer_desc")
	}
	bgpPeerMetricPrefix := "bgp_peer"

	bgpDesc = map[string]*prometheus.Desc{
		"ribCount":        colPromDesc(bgpSubsystem, "rib_count_total", "Number of routes in the RIB.", bgpLabels),
		"ribMemory":       colPromDesc(bgpSubsystem, "rib_memory_bytes", "Memory consumbed by the RIB.", bgpLabels),
		"peerCount":       colPromDesc(bgpSubsystem, "peers_count_total", "Number peers configured.", bgpLabels),
		"peerMemory":      colPromDesc(bgpSubsystem, "peers_memory_bytes", "Memory consumed by peers.", bgpLabels),
		"peerGroupCount":  colPromDesc(bgpSubsystem, "peer_groups_count_total", "Number of peer groups configured.", bgpLabels),
		"peerGroupMemory": colPromDesc(bgpSubsystem, "peer_groups_memory_bytes", "Memory consumed by peer groups.", bgpLabels),

		"msgRcvd":               colPromDesc(bgpPeerMetricPrefix, "message_received_total", "Number of received messages.", bgpPeerLabels),
		"msgSent":               colPromDesc(bgpPeerMetricPrefix, "message_sent_total", "Number of sent messages.", bgpPeerLabels),
		"prefixReceivedCount":   colPromDesc(bgpPeerMetricPrefix, "prefixes_received_count_total", "Number of prefixes received.", bgpPeerLabels),
		"prefixAdvertisedCount": colPromDesc(bgpPeerMetricPrefix, "prefixes_advertised_count_total", "Number of prefixes advertised.", bgpPeerLabels),
		"state":                 colPromDesc(bgpPeerMetricPrefix, "state", "State of the peer (2 = Administratively Down, 1 = Established, 0 = Down).", bgpPeerLabels),
		"UptimeSec":             colPromDesc(bgpPeerMetricPrefix, "uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":           colPromDesc(bgpPeerMetricPrefix, "types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}

	bgpL2vpnLabels := []string{"vni", "type", "vxlanIf", "tenantVrf"}
	bgpL2vpnMetricPrefix := "bgp_l2vpn_evpn"

	bgpL2vpnDesc = map[string]*prometheus.Desc{
		"numMacs":        colPromDesc(bgpL2vpnMetricPrefix, "mac_count_total", "Number of known MAC addresses", bgpL2vpnLabels),
		"numArpNd":       colPromDesc(bgpL2vpnMetricPrefix, "arp_nd_count_total", "Number of ARP / ND entries", bgpL2vpnLabels),
		"numRemoteVteps": colPromDesc(bgpL2vpnMetricPrefix, "remote_vtep_count_total", "Number of known remote VTEPs. A value of -1 indicates a non-integer output from FRR, such as n/a.", bgpL2vpnLabels),
	}

}

func collectBGP(ch chan<- prometheus.Metric, AFI string, logger log.Logger) error {
	setBgpDesc()
	SAFI := ""

	if (AFI == "ipv4") || (AFI == "ipv6") {
		SAFI = "unicast"

	} else if AFI == "l2vpn" {
		SAFI = "evpn"
	}

	jsonBGPSum, err := getBGPSummary(AFI, SAFI)
	if err != nil {
		return fmt.Errorf("cannot get bgp %s %s summary: %s", AFI, SAFI, err)
	} else {
		if err := processBGPSummary(ch, jsonBGPSum, AFI, SAFI, logger); err != nil {
			return err
		}
	}
	return nil
}

func getBGPSummary(AFI string, SAFI string) ([]byte, error) {
	args := []string{"-c", fmt.Sprintf("show bgp vrf all %s %s summary json", AFI, SAFI)}

	return execVtyshCommand(args...)
}

func processBGPSummary(ch chan<- prometheus.Metric, jsonBGPSum []byte, AFI string, SAFI string, logger log.Logger) error {
	var jsonMap map[string]bgpProcess
	if err := json.Unmarshal(jsonBGPSum, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal bgp summary json: %s", err)
	}

	var peerDescJSON map[string]map[string]string
	var peerDescText map[string]string
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
		localAs := strconv.FormatInt(vrfData.AS, 10)
		procLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs}
		// No point collecting metrics if no peers configured.
		if vrfData.PeerCount != 0 {
			newGauge(ch, bgpDesc["ribCount"], vrfData.RIBCount, procLabels...)
			newGauge(ch, bgpDesc["ribMemory"], vrfData.RIBMemory, procLabels...)
			newGauge(ch, bgpDesc["peerCount"], vrfData.PeerCount, procLabels...)
			newGauge(ch, bgpDesc["peerMemory"], vrfData.PeerMemory, procLabels...)
			newGauge(ch, bgpDesc["peerGroupCount"], vrfData.PeerGroupCount, procLabels...)
			newGauge(ch, bgpDesc["peerGroupMemory"], vrfData.PeerGroupMemory, procLabels...)

			for peerIP, peerData := range vrfData.Peers {
				// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as"
				peerLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs, peerIP, strconv.FormatInt(peerData.RemoteAs, 10)}

				if *bgpPeerDescs {
					d := ""
					if *bgpPeerDescsText {
						d = peerDescText[peerIP]
					} else {
						d = peerDescJSON[peerIP]["desc"]
					}
					// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as", "peer_desc"
					peerLabels = append(peerLabels, d)
				}

				// In earlier versions of FRR did not expose a summary of advertised prefixes for all peers, but in later versions it can get with PfxSnt field.
				if peerData.PfxSnt != nil {
					newGauge(ch, bgpDesc["prefixAdvertisedCount"], *peerData.PfxSnt, peerLabels...)
				} else if *bgpAdvertisedPrefixes {
					wgAdvertisedPrefixes.Add(1)
					go getPeerAdvertisedPrefixes(ch, wgAdvertisedPrefixes, AFI, SAFI, vrfName, peerIP, logger, peerLabels...)
				}

				newCounter(ch, bgpDesc["msgRcvd"], peerData.MsgRcvd, peerLabels...)
				newCounter(ch, bgpDesc["msgSent"], peerData.MsgSent, peerLabels...)
				newGauge(ch, bgpDesc["UptimeSec"], peerData.PeerUptimeMsec*0.001, peerLabels...)

				// In earlier versions of FRR, the prefixReceivedCount JSON element is used for the number of recieved prefixes, but in later versions it was changed to PfxRcd.
				prefixReceived := 0.0
				if peerData.PrefixReceivedCount != 0 {
					prefixReceived = peerData.PrefixReceivedCount
				} else if peerData.PfxRcd != 0 {
					prefixReceived = peerData.PfxRcd
				}
				newGauge(ch, bgpDesc["prefixReceivedCount"], prefixReceived, peerLabels...)

				if *bgpPeerTypes {
					for _, descKey := range *frrBGPDescKey {
						if peerDescJSON[peerIP][descKey] != "" {
							if _, exist := peerTypes[strings.TrimSpace(peerDescJSON[peerIP][descKey])]; !exist {
								peerTypes[strings.TrimSpace(peerDescJSON[peerIP][descKey])] = 0
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
							if peerDescJSON[peerIP][descKey] != "" {
								peerTypes[strings.TrimSpace(peerDescJSON[peerIP][descKey])]++
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

func getPeerAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, AFI string, SAFI string, vrfName string, neighbor string, logger log.Logger, peerLabels ...string) {
	defer wg.Done()

	var cmd string
	if strings.ToLower(vrfName) == "default" {
		cmd = fmt.Sprintf("show bgp  %s %s neighbors %s advertised-routes json", AFI, SAFI, neighbor)
	} else {
		cmd = fmt.Sprintf("show bgp vrf %s %s %s neighbors %s advertised-routes json", vrfName, AFI, SAFI, neighbor)
	}

	output, err := execVtyshCommand("-c", cmd)
	if err != nil {
		level.Error(logger).Log("msg", "get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	var advertisedPrefixes bgpAdvertisedRoutes
	if err := json.Unmarshal(output, &advertisedPrefixes); err != nil {
		level.Error(logger).Log("msg", "get neighbor advertised prefixes failed", "afi", AFI, "safi", SAFI, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	newGauge(ch, bgpDesc["prefixAdvertisedCount"], advertisedPrefixes.TotalPrefixCounter, peerLabels...)
}

type bgpProcess struct {
	RouterID        string
	AS              int64
	RIBCount        float64
	RIBMemory       float64
	PeerCount       float64
	PeerMemory      float64
	PeerGroupCount  float64
	PeerGroupMemory float64
	Peers           map[string]*bgpPeerSession
}

type bgpPeerSession struct {
	State               string
	RemoteAs            int64
	MsgRcvd             float64
	MsgSent             float64
	PeerUptimeMsec      float64
	PrefixReceivedCount float64
	PfxRcd              float64
	PfxSnt              *float64
}
type bgpAdvertisedRoutes struct {
	TotalPrefixCounter float64 `json:"totalPrefixCounter"`
}

// Returns:
//  - Map from JSON formatted BGP peer descriptions
//  - Plain text description of peers
//  - Error
func getBGPPeerDesc(logger log.Logger) (map[string]map[string]string, map[string]string, error) {
	descJSON := make(map[string]map[string]string)
	descText := make(map[string]string)

	output, err := execVtyshCommand("-c", "show run bgpd")
	if err != nil {
		return nil, nil, err
	}
	r := regexp.MustCompile(`.*neighbor (.*) description (.*)\n`)
	matches := r.FindAllStringSubmatch(string(output), -1)
	for _, match := range matches {
		if !*bgpPeerDescsText {
			var peerDesc map[string]string
			if err := json.Unmarshal([]byte(match[2]), &peerDesc); err != nil {
				// Don't return an error as unmarshalling is best effort.
				level.Error(logger).Log("msg", "cannot unmarshall bgp description", "description", match[2], "neighbor", match[1], "err", err)
			}
			descJSON[match[1]] = peerDesc
		}
		descText[match[1]] = match[2]
	}
	return descJSON, descText, nil
}
