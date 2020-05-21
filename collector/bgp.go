package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	bgpSubsystem        = "bgp"
	bgpPeerMetricPrefix = "bgp_peer"

	bgpDesc = map[string]*prometheus.Desc{}

	bgpErrors           = []error{}
	totalBGPErrors      = 0.0
	bgp6Errors          = []error{}
	totalBGP6Errors     = 0.0
	bgpL2VPNErrors      = []error{}
	totalBGPL2VPNErrors = 0.0

	bgpPeerTypes          = kingpin.Flag("collector.bgp.peer-types", "Enable scraping of BGP peer types from peer descriptions (default: disabled).").Default("False").Bool()
	bgpPeerDescs          = kingpin.Flag("collector.bgp.peer-descriptions", "Add the BGP peer description as a label to peer metrics. (default: disabled).").Default("False").Bool()
	bgpPeerDescsText      = kingpin.Flag("collector.bgp.peer-descriptions.plain-text", "Use the full text field of the BGP peer description, instead of a JSON formatted description (default: disabled).").Default("False").Bool()
	bgpAdvertisedPrefixes = kingpin.Flag("collector.bgp.advertised-prefixes", "Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer (default: disabled).").Default("False").Bool()
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
	if len(bgpDesc) == 0 {
		setDesc()
	}
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
	return false
}

// Describe implemented as per the prometheus.Collector interface.
func (*BGP6Collector) Describe(ch chan<- *prometheus.Desc) {
	if len(bgpDesc) == 0 {
		setDesc()
	}
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
	if len(bgpDesc) == 0 {
		setDesc()
	}
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

func setDesc() {
	bgpLabels := []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerTypeLabels := []string{"type", "afi", "safi"}
	bgpPeerLabels := append(bgpLabels, "peer", "peer_as")

	if *bgpPeerDescs {
		bgpPeerLabels = append(bgpLabels, "peer", "peer_as", "peer_desc")
	}

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
		"state":                 colPromDesc(bgpPeerMetricPrefix, "state", "State of the peer (1 = Established, 0 = Down).", bgpPeerLabels),
		"UptimeSec":             colPromDesc(bgpPeerMetricPrefix, "uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":           colPromDesc(bgpPeerMetricPrefix, "types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}
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
			errors = append(errors, err)
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

	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, vtyshPath, args...).Output()
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

	var peerDescJSON map[string]bgpPeerDesc
	var peerDescText map[string]string
	var err error
	if *bgpPeerTypes || *bgpPeerDescs {
		peerDescJSON, peerDescText, err = getBGPPeerDesc()
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

				if *bgpAdvertisedPrefixes {
					wgAdvertisedPrefixes.Add(1)
					go getPeerAdvertisedPrefixes(ch, wgAdvertisedPrefixes, AFI, SAFI, vrfName, peerIP, peerLabels...)
				}

				if *bgpPeerDescs {
					d := ""
					if *bgpPeerDescsText {
						if desc, exists := peerDescText[peerIP]; exists {
							d = desc
						}
					} else {
						if desc, exists := peerDescJSON[peerIP]; exists {
							d = desc.Desc
						}
					}
					// The labels are "vrf", "afi", "safi", "local_as", "peer", "remote_as", "peer_desc"
					peerLabels = append(peerLabels, d)
				}
				newCounter(ch, bgpDesc["msgRcvd"], peerData.MsgRcvd, peerLabels...)
				newCounter(ch, bgpDesc["msgSent"], peerData.MsgSent, peerLabels...)
				newGauge(ch, bgpDesc["prefixReceivedCount"], peerData.PrefixReceivedCount, peerLabels...)
				newGauge(ch, bgpDesc["UptimeSec"], peerData.PeerUptimeMsec*0.001, peerLabels...)

				peerState := 0.0
				if strings.ToLower(peerData.State) == "established" {
					peerState = 1
					if *bgpPeerTypes {
						if desc, exists := peerDescJSON[peerIP]; exists {
							if desc.Type != "" {
								if _, exists := peerTypes[strings.TrimSpace(desc.Type)]; exists {
									peerTypes[strings.TrimSpace(desc.Type)]++
								} else {
									peerTypes[strings.TrimSpace(desc.Type)] = 1
								}
							}
						}
					}
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

func getPeerAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, AFI string, SAFI string, vrfName string, neighbor string, peerLabels ...string) {
	defer wg.Done()

	errors := []error{}
	totalErrors := 0.0

	args := []string{}
	if strings.ToLower(vrfName) == "default" {
		args = []string{"-c", fmt.Sprintf("show bgp  %s %s neighbors %s advertised-routes json", AFI, SAFI, neighbor)}
	} else {
		args = []string{"-c", fmt.Sprintf("show bgp vrf %s %s %s neighbors %s advertised-routes json", vrfName, AFI, SAFI, neighbor)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, vtyshPath, args...).Output()
	if err != nil {
		totalErrors++
		errors = append(errors, err)
	}

	var advertisedPrefixes bgpAdvertisedRoutes
	if err := json.Unmarshal(output, &advertisedPrefixes); err != nil {
		totalErrors++
		errors = append(errors, err)
	}

	if AFI == "ipv4" {
		bgpErrors = append(bgpErrors, errors...)
		if totalErrors > 0 {
			totalBGPErrors += totalErrors
			return
		}
	} else if AFI == "ipv6" {
		bgp6Errors = append(bgpErrors, errors...)
		if totalErrors > 0 {
			totalBGP6Errors += totalErrors
			return
		}
	} else if AFI == "l2vpn" {
		bgpL2VPNErrors = append(bgpErrors, errors...)
		if totalErrors > 0 {
			totalBGPL2VPNErrors += totalErrors
			return
		}
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
}
type bgpAdvertisedRoutes struct {
	TotalPrefixCounter float64 `json:"totalPrefixCounter"`
}

// Returns:
//  - JSON formatted description of peers
//  - Plain text description of peers
//  - Error
func getBGPPeerDesc() (map[string]bgpPeerDesc, map[string]string, error) {
	args := []string{"-c", "show run bgpd"}
	descJSON := make(map[string]bgpPeerDesc)
	descText := make(map[string]string)

	ctx, cancel := context.WithTimeout(context.Background(), vtyshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, vtyshPath, args...).Output()
	if err != nil {
		return nil, nil, err
	}
	r := regexp.MustCompile(`.*neighbor (.*) description (.*)\n`)
	matches := r.FindAllStringSubmatch(string(output), -1)
	for _, match := range matches {
		var peerDesc bgpPeerDesc
		if err := json.Unmarshal([]byte(match[2]), &peerDesc); err != nil {
			// Don't return an error if scraping fails as description unmarshalling is best effort.
			bgpErrors = append(bgpErrors, fmt.Errorf("cannot unmarshall bgp description %s for peer %s : %s", match[2], match[1], err))
		}
		descJSON[match[1]] = peerDesc
		descText[match[1]] = match[2]
	}
	return descJSON, descText, nil
}

type bgpPeerDesc struct {
	Type string `json:"type"`
	Desc string `json:"desc"`
}
