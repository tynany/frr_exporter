package collector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	bgpSubsystem        = "bgp"
	bgpPeerMetricPrefix = "bgp_peer"

	bgpLabels         = []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerLabels     = append(bgpLabels, "peer", "peer_as")
	bgpPeerTypeLabels = []string{"type", "afi", "safi"}

	bgpDesc = map[string]*prometheus.Desc{
		"ribCount":        colPromDesc(bgpSubsystem, "rib_count_total", "Number of routes in the RIB.", bgpLabels),
		"ribMemory":       colPromDesc(bgpSubsystem, "rib_memory_bytes", "Memory consumbed by the RIB.", bgpLabels),
		"peerCount":       colPromDesc(bgpSubsystem, "peers_count_total", "Number peers configured.", bgpLabels),
		"peerMemory":      colPromDesc(bgpSubsystem, "peers_memory_bytes", "Memory consumed by peers.", bgpLabels),
		"peerGroupCount":  colPromDesc(bgpSubsystem, "peer_groups_count_total", "Number of peer groups configured.", bgpLabels),
		"peerGroupMemory": colPromDesc(bgpSubsystem, "peer_groups_memory_bytes", "Memory consumed by peer groups.", bgpLabels),

		"msgRcvd":             colPromDesc(bgpPeerMetricPrefix, "message_received_total", "Number of received messages.", bgpPeerLabels),
		"msgSent":             colPromDesc(bgpPeerMetricPrefix, "message_sent_total", "Number of sent messages.", bgpPeerLabels),
		"prefixReceivedCount": colPromDesc(bgpPeerMetricPrefix, "prefixes_received_count_total", "Number active prefixes received.", bgpPeerLabels),
		"state":               colPromDesc(bgpPeerMetricPrefix, "state", "State of the peer (1 = Established, 0 = Down).", bgpPeerLabels),
		"UptimeSec":           colPromDesc(bgpPeerMetricPrefix, "uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":         colPromDesc(bgpPeerMetricPrefix, "types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}

	bgpErrors           = []error{}
	totalBGPErrors      = 0.0
	bgp6Errors          = []error{}
	totalBGP6Errors     = 0.0
	bgpL2VPNErrors      = []error{}
	totalBGPL2VPNErrors = 0.0

	bgpPeerTypes = kingpin.Flag("collector.bgp.peer-types", "Enable scraping of BGP peer types from peer descriptions (default: disabled).").Default("False").Bool()
)

// BGPCollector collects BGP metrics, implemented as per prometheus.Collector interface.
type BGPCollector struct{}

// NewBGPCollector returns a BGPCollector struct.
func NewBGPCollector() *BGPCollector {
	return &BGPCollector{}
}

// Name of the collector. Used to populate flag name.
func (*BGPCollector) Name() string {
	return bgpSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*BGPCollector) Help() string {
	return "Collect BGP Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*BGPCollector) EnabledByDefault() bool {
	return true
}

// Describe implemented as per the prometheus.Collector interface.
func (*BGPCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range bgpDesc {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *BGPCollector) Collect(ch chan<- prometheus.Metric) {
	collectBGP(ch, "ipv4")
}

// CollectErrors returns what errors have been gathered.
func (*BGPCollector) CollectErrors() []error {
	return bgpErrors
}

// CollectTotalErrors returns total errors.
func (*BGPCollector) CollectTotalErrors() float64 {
	return totalBGPErrors
}

// BGP6Collector collects BGP metrics, implemented as per prometheus.Collector interface.
type BGP6Collector struct{}

// NewBGP6Collector returns a BGP6Collector struct.
func NewBGP6Collector() *BGP6Collector {
	return &BGP6Collector{}
}

// Name of the collector. Used to populate flag name.
func (*BGP6Collector) Name() string {
	return bgpSubsystem + "6"
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*BGP6Collector) Help() string {
	return "Collect BGP IPv6 Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*BGP6Collector) EnabledByDefault() bool {
	return true
}

// Describe implemented as per the prometheus.Collector interface.
func (*BGP6Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range bgpDesc {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *BGP6Collector) Collect(ch chan<- prometheus.Metric) {
	collectBGP(ch, "ipv6")
}

// CollectErrors returns what errors have been gathered.
func (*BGP6Collector) CollectErrors() []error {
	return bgp6Errors
}

// CollectTotalErrors returns total errors.
func (*BGP6Collector) CollectTotalErrors() float64 {
	return totalBGP6Errors
}

// BGPL2VPNCollector collects BGP metrics, implemented as per prometheus.Collector interface.
type BGPL2VPNCollector struct{}

// NewBGPL2VPNCollector returns a BGPL2VPNCollector struct.
func NewBGPL2VPNCollector() *BGPL2VPNCollector {
	return &BGPL2VPNCollector{}
}

// Name of the collector. Used to populate flag name.
func (*BGPL2VPNCollector) Name() string {
	return bgpSubsystem + "l2vpn"
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*BGPL2VPNCollector) Help() string {
	return "Collect BGP L2VPN Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*BGPL2VPNCollector) EnabledByDefault() bool {
	return false
}

// Describe implemented as per the prometheus.Collector interface.
func (*BGPL2VPNCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range bgpDesc {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *BGPL2VPNCollector) Collect(ch chan<- prometheus.Metric) {
	collectBGP(ch, "l2vpn")
}

// CollectErrors returns what errors have been gathered.
func (*BGPL2VPNCollector) CollectErrors() []error {
	return bgpL2VPNErrors
}

// CollectTotalErrors returns total errors.
func (*BGPL2VPNCollector) CollectTotalErrors() float64 {
	return totalBGPL2VPNErrors
}

func collectBGP(ch chan<- prometheus.Metric, AFI string) {
	SAFI := ""
	errors := []error{}
	totalErrors := 0.0

	if (AFI == "ipv4") || (AFI == "ipv6") {
		SAFI = "unicast"

	} else if AFI == "l2vpn" {
		SAFI = "evpn"
	}

	jsonBGPSum, err := getBGPSummary(AFI, SAFI)
	if err != nil {
		totalErrors++
		errors = append(errors, fmt.Errorf("cannot get bgp %s %s summary: %s", AFI, SAFI, err))
	} else {
		if err := processBGPSummary(ch, jsonBGPSum, AFI, SAFI); err != nil {
			totalErrors++
			errors = append(errors, fmt.Errorf("%s", err))
		}
	}

	if AFI == "ipv4" {
		bgpErrors = errors
		if totalErrors > 0 {
			totalBGPErrors += totalErrors
		}
	} else if AFI == "ipv6" {
		bgp6Errors = errors
		if totalErrors > 0 {
			totalBGP6Errors += totalErrors
		}
	} else if AFI == "l2vpn" {
		bgpL2VPNErrors = errors
		if totalErrors > 0 {
			totalBGPL2VPNErrors += totalErrors
		}
	}

}

func getBGPSummary(AFI string, SAFI string) ([]byte, error) {
	args := []string{"-c", fmt.Sprintf("show bgp vrf all %s %s summary json", AFI, SAFI)}
	output, err := exec.Command(vtyshPath, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func processBGPSummary(ch chan<- prometheus.Metric, jsonBGPSum []byte, AFI string, SAFI string) error {
	var jsonMap map[string]bgpProcess

	if err := json.Unmarshal(jsonBGPSum, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal bgp summary json: %s", err)
	}

	var bgpPeerDesc map[string]string
	var err error
	if *bgpPeerTypes == true {
		bgpPeerDesc, err = getBGPPeerDesc()
		if err != nil {
			return err
		}
	}

	peerTypes := make(map[string]float64)

	for vrfName, vrfData := range jsonMap {
		// The labels are "vrf", "afi",  "safi", "local_as"
		localAs := strconv.FormatInt(vrfData.AS, 10)
		bgpProcLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs}
		// No point collecting metrics if no peers configured.
		if vrfData.PeerCount != 0 {

			newGauge(ch, bgpDesc["ribCount"], vrfData.RIBCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["ribMemory"], vrfData.RIBMemory, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerCount"], vrfData.PeerCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerMemory"], vrfData.PeerMemory, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerGroupCount"], vrfData.PeerGroupCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerGroupMemory"], vrfData.PeerGroupMemory, bgpProcLabels...)

			for peerIP, peerData := range vrfData.Peers {
				// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as"
				bgpPeerLabels := []string{strings.ToLower(vrfName), strings.ToLower(AFI), strings.ToLower(SAFI), localAs, peerIP, strconv.FormatInt(peerData.RemoteAs, 10)}

				newCounter(ch, bgpDesc["msgRcvd"], peerData.MsgRcvd, bgpPeerLabels...)
				newCounter(ch, bgpDesc["msgSent"], peerData.MsgSent, bgpPeerLabels...)
				newGauge(ch, bgpDesc["prefixReceivedCount"], peerData.PrefixReceivedCount, bgpPeerLabels...)
				newGauge(ch, bgpDesc["UptimeSec"], peerData.PeerUptimeMsec*0.001, bgpPeerLabels...)

				peerState := 0.0
				if strings.ToLower(peerData.State) == "established" {
					peerState = 1
					if *bgpPeerTypes == true {
						if desc, exists := bgpPeerDesc[peerIP]; exists {
							var peerType bgpPeerType
							if err := json.Unmarshal([]byte(desc), &peerType); err != nil {
								goto NoPeerType
							}

							if peerType.Type != "" {
								if _, exists := peerTypes[strings.TrimSpace(peerType.Type)]; exists {
									peerTypes[strings.TrimSpace(peerType.Type)]++
								} else {
									peerTypes[strings.TrimSpace(peerType.Type)] = 1
								}
							}
						}
					}
				}
			NoPeerType:
				newGauge(ch, bgpDesc["state"], peerState, bgpPeerLabels...)
			}
		}
	}

	for peerType, count := range peerTypes {
		peerTypeLabels := []string{peerType, strings.ToLower(AFI), strings.ToLower(SAFI)}
		newGauge(ch, bgpDesc["peerTypesUp"], count, peerTypeLabels...)
	}
	return nil
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
}

func getBGPPeerDesc() (map[string]string, error) {
	args := []string{"-c", "show run bgpd"}
	desc := make(map[string]string)
	output, err := exec.Command(vtyshPath, args...).Output()
	if err != nil {
		return nil, err
	}
	r := regexp.MustCompile(`.*neighbor (.*) description (.*)\n`)
	matches := r.FindAllStringSubmatch(string(output), -1)
	for i := range matches {
		desc[matches[i][1]] = matches[i][2]
	}
	return desc, nil
}

type bgpPeerType struct {
	Type string `json:"type"`
}
