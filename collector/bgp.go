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
	bgpSubsystem = "bgp"

	bgpLabels         = []string{"vrf", "address_family", "local_as"}
	bgpPeerLabels     = append(bgpLabels, "peer", "peer_as")
	bgpPeerTypeLabels = []string{"type", "address_family"}

	bgpDesc = map[string]*prometheus.Desc{
		"ribEntries":       colPromDesc(bgpSubsystem, "rib_entries", "Number of routes in the RIB.", bgpLabels),
		"ribMemUsgage":     colPromDesc(bgpSubsystem, "rib_memory_usage_bytes", "Memory consumbed by the RIB.", bgpLabels),
		"peerTotal":        colPromDesc(bgpSubsystem, "peers", "Number peers configured.", bgpLabels),
		"peerMemUsage":     colPromDesc(bgpSubsystem, "peers_memory_usage_bytes", "Memory consumed by peers.", bgpLabels),
		"peerGrps":         colPromDesc(bgpSubsystem, "peer_groups", "Number of peer groups configured.", bgpLabels),
		"peerGrpsMemUsage": colPromDesc(bgpSubsystem, "peer_groups_memory_bytes", "Memory consumed by peer groups.", bgpLabels),

		"peerMsgIn":     colPromDesc(bgpSubsystem, "message_input_total", "Number of received messages.", bgpPeerLabels),
		"peerMsgOut":    colPromDesc(bgpSubsystem, "message_output_total", "Number of sent messages.", bgpPeerLabels),
		"peerPrfAct":    colPromDesc(bgpSubsystem, "prefixes_active", "Number of active prefixes.", bgpPeerLabels),
		"peerUp":        colPromDesc(bgpSubsystem, "peer_up", "State of the peer (1 = Established, 0 = Down).", bgpPeerLabels),
		"peerUptimeSec": colPromDesc(bgpSubsystem, "peer_uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":   colPromDesc(bgpSubsystem, "peer_types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}

	bgpErrors       = []error{}
	totalBGPErrors  = 0.0
	bgp6Errors      = []error{}
	totalBGP6Errors = 0.0

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

func collectBGP(ch chan<- prometheus.Metric, addressFamily string) {
	errors := []error{}
	totalErrors := 0.0

	afMod := "unicast"

	jsonBGPSum, err := getBGPSummary(addressFamily, afMod)
	if err != nil {
		totalErrors++
		errors = append(errors, fmt.Errorf("cannot get bgp %s %s summary: %s", addressFamily, afMod, err))
	} else {
		if err := processBGPSummary(ch, jsonBGPSum, addressFamily+afMod); err != nil {
			totalErrors++
			errors = append(errors, fmt.Errorf("%s", err))
		}
	}

	if totalErrors > 0 {
		if addressFamily == "ipv4" {
			totalBGPErrors = totalBGPErrors + totalErrors
		} else if addressFamily == "ipv6" {
			totalBGP6Errors = totalBGP6Errors + totalErrors
		}
	}

	if addressFamily == "ipv4" {
		bgpErrors = errors
	} else if addressFamily == "ipv6" {
		bgp6Errors = errors
	}
}

func getBGPSummary(addressFamily string, addressFamilyModifier string) ([]byte, error) {
	args := []string{"-c", fmt.Sprintf("show ip bgp vrf all %s %s summary json", addressFamily, addressFamilyModifier)}
	output, err := exec.Command(vtyshPath, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func processBGPSummary(ch chan<- prometheus.Metric, jsonBGPSum []byte, addressFamily string) error {
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
		// The labels are "vrf", "address_family", "local_as"
		localAs := strconv.FormatInt(vrfData.AS, 10)
		bgpProcLabels := []string{strings.ToLower(vrfName), strings.ToLower(addressFamily), localAs}
		// No point collecting metrics if no peers configured.
		if vrfData.PeerCount != 0 {

			newGauge(ch, bgpDesc["ribEntries"], vrfData.RIBCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["ribMemUsgage"], vrfData.RIBMemory, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerTotal"], vrfData.PeerCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerMemUsage"], vrfData.PeerMemory, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerGrps"], vrfData.PeerGroupCount, bgpProcLabels...)
			newGauge(ch, bgpDesc["peerGrpsMemUsage"], vrfData.PeerGroupMemory, bgpProcLabels...)

			for peerIP, peerData := range vrfData.Peers {
				// The labels are "vrf", "address_family", "local_as", "peer", "remote_as"
				bgpPeerLabels := []string{strings.ToLower(vrfName), strings.ToLower(addressFamily), localAs, peerIP, strconv.FormatInt(peerData.RemoteAs, 10)}

				newCounter(ch, bgpDesc["peerMsgIn"], peerData.MsgRcvd, bgpPeerLabels...)
				newCounter(ch, bgpDesc["peerMsgOut"], peerData.MsgSent, bgpPeerLabels...)
				newGauge(ch, bgpDesc["peerPrfAct"], peerData.PrefixReceivedCount, bgpPeerLabels...)
				newGauge(ch, bgpDesc["peerUptimeSec"], peerData.PeerUptimeMsec*0.001, bgpPeerLabels...)

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
				newGauge(ch, bgpDesc["peerUp"], peerState, bgpPeerLabels...)
			}
		}
	}

	for peerType, count := range peerTypes {
		peerTypeLabels := []string{peerType, strings.ToLower(addressFamily)}
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
