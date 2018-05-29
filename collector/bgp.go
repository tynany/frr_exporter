package collector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	bgpSubsystem = "bgp"

	bgpLabels     = []string{"vrf", "address_family"}
	bgpPeerLabels = append(bgpLabels, "peer")

	bgpRibEntries       = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "rib_entries"), "Number of routes in the RIB.", bgpLabels, nil)
	bgpRibMemUsgage     = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "rib_memory_usage_bytes"), "Memory consumbed by the RIB.", bgpLabels, nil)
	bgpPeerTotal        = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peers"), "Number peers configured.", bgpLabels, nil)
	bgpPeerMemUsage     = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peers_memory_usage_bytes"), "Memory consumed by peers.", bgpLabels, nil)
	bgpPeerGrps         = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peer_groups"), "Number of peer groups configured.", bgpLabels, nil)
	bgpPeerGrpsMemUsage = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peer_groups_memory_bytes"), "Memory consumed by peer groups.", bgpLabels, nil)

	bgpPeerMsgIn     = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "message_input_total"), "Number of received messages.", bgpPeerLabels, nil)
	bgpPeerMsgOut    = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "message_output_total"), "Number of sent messages.", bgpPeerLabels, nil)
	bgpPeerPrfAct    = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "prefixes_active"), "Number of active prefixes.", bgpPeerLabels, nil)
	bgpPeerUp        = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peer_up"), "State of the peer (1 = Established, 0 = Down).", bgpPeerLabels, nil)
	bgpPeerUptimeSec = prometheus.NewDesc(prometheus.BuildFQName(namespace, bgpSubsystem, "peer_uptime_seconds"), "How long has the peer been up.", bgpPeerLabels, nil)

	bgpErrors = []error{}
)

// BGPCollector collects BGP metrics, implemented as per prometheus.Collector interface.
type BGPCollector struct {
	FRRVTYSHPath string
}

// type BGPTST struct {
// }

func (*BGPCLIHelper) NewCollector(path string, name string) *Collector {
	return &Collector{
		Name:          name,
		PromCollector: NewBGPCollector(path),
		Errors:        new(BGPErrorCollector),
	}
}

// BGPCLIHelper is used to populate flags.
type BGPCLIHelper struct {
}

// BGPErrorCollector collects errors collecting BGP metrics.
type BGPErrorCollector struct{}

// NewBGPCollector returns a BGPCollector struct with the FRRVTYSHPath populated.
func NewBGPCollector(path string) prometheus.Collector {
	return &BGPCollector{FRRVTYSHPath: path}
}

// Describe implemented as per the prometheus.Collector interface.
func (*BGPCollector) Describe(ch chan<- *prometheus.Desc) {

	ch <- bgpRibEntries
	ch <- bgpRibMemUsgage
	ch <- bgpPeerTotal
	ch <- bgpPeerMemUsage
	ch <- bgpPeerGrps
	ch <- bgpPeerGrpsMemUsage
	ch <- bgpPeerMsgIn
	ch <- bgpPeerMsgOut
	ch <- bgpPeerPrfAct
	ch <- bgpPeerUp

}

// Name of the collector. Used to populate flag name.
func (*BGPCLIHelper) Name() string {
	return bgpSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*BGPCLIHelper) Help() string {
	return "Collect BGP Metrics."
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*BGPCLIHelper) EnabledByDefault() bool {
	return true
}

// Collect implemented as per the prometheus.Collector interface.
func (c *BGPCollector) Collect(ch chan<- prometheus.Metric) {

	addressFamilies := []string{"ipv4", "ipv6"}
	addressFamilyModifiers := []string{"unicast"}

	for _, af := range addressFamilies {
		for _, afMod := range addressFamilyModifiers {
			jsonBGPSum, err := getBGPSummary(af, afMod, c.FRRVTYSHPath)
			if err != nil {
				bgpErrors = append(bgpErrors, fmt.Errorf("cannot get bgp %s %s summary: %s", af, afMod, err))
			} else {
				if err := processBGPSummary(ch, jsonBGPSum, af+afMod); err != nil {
					bgpErrors = append(bgpErrors, fmt.Errorf("%s", err))
				}
			}
		}
	}
}

// CollectErrors returns what errors have been gathered.
func (*BGPErrorCollector) CollectErrors() []error {
	return bgpErrors
}

type bgpProcess struct {
	RouterID        string
	AS              int
	RIBCount        int
	RIBMemory       int
	PeerCount       int
	PeerMemory      int
	PeerGroupCount  int
	PeerGroupMemory int
	Peers           map[string]*bgpPeerSession
}

type bgpPeerSession struct {
	State               string
	MsgRcvd             int
	MsgSent             int
	PeerUptimeMsec      int64
	PrefixReceivedCount int
}

func getBGPSummary(addressFamily string, addressFamilyModifier string, frrVTYSHPath string) ([]byte, error) {
	args := []string{"-c", fmt.Sprintf("show ip bgp vrf all %s %s summary json", addressFamily, addressFamilyModifier)}
	output, err := exec.Command(frrVTYSHPath, args...).Output()
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

	for vrfName, vrfData := range jsonMap {
		// The labels are "vrf", "address_family",
		bgpProcLabels := []string{vrfName, addressFamily}
		// No point collecting metrics if no peers configured.
		if vrfData.PeerCount != 0 {
			ch <- prometheus.MustNewConstMetric(bgpRibEntries, prometheus.GaugeValue, float64(vrfData.RIBCount), bgpProcLabels...)
			ch <- prometheus.MustNewConstMetric(bgpRibMemUsgage, prometheus.GaugeValue, float64(vrfData.RIBMemory), bgpProcLabels...)
			ch <- prometheus.MustNewConstMetric(bgpPeerTotal, prometheus.GaugeValue, float64(vrfData.PeerCount), bgpProcLabels...)
			ch <- prometheus.MustNewConstMetric(bgpPeerMemUsage, prometheus.GaugeValue, float64(vrfData.PeerMemory), bgpProcLabels...)
			ch <- prometheus.MustNewConstMetric(bgpPeerGrps, prometheus.GaugeValue, float64(vrfData.PeerGroupCount), bgpProcLabels...)
			ch <- prometheus.MustNewConstMetric(bgpPeerGrpsMemUsage, prometheus.GaugeValue, float64(vrfData.PeerGroupMemory), bgpProcLabels...)

			for peerIP, peerData := range vrfData.Peers {
				// The labels are "vrf", "address_family", "peer"
				bgpPeerLabels := []string{vrfName, addressFamily, peerIP}

				ch <- prometheus.MustNewConstMetric(bgpPeerMsgIn, prometheus.CounterValue, float64(peerData.MsgRcvd), bgpPeerLabels...)
				ch <- prometheus.MustNewConstMetric(bgpPeerMsgOut, prometheus.CounterValue, float64(peerData.MsgSent), bgpPeerLabels...)
				ch <- prometheus.MustNewConstMetric(bgpPeerPrfAct, prometheus.GaugeValue, float64(peerData.PrefixReceivedCount), bgpPeerLabels...)
				ch <- prometheus.MustNewConstMetric(bgpPeerUptimeSec, prometheus.GaugeValue, float64(peerData.PeerUptimeMsec)*0.001, bgpPeerLabels...)

				peerState := 0.0
				if strings.ToLower(peerData.State) == "established" {
					peerState = 1
				}
				ch <- prometheus.MustNewConstMetric(bgpPeerUp, prometheus.GaugeValue, peerState, bgpPeerLabels...)
			}
		}
	}
	return nil
}
