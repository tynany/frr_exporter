package collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	vrrpStatusInitialize string = "Initialize"
	vrrpStatusBackup            = "Backup"
	vrrpStatusMaster            = "Master"
)

var (
	vrrpSubsystem = "vrrp"
	vrrpDesc      map[string]*prometheus.Desc

	vrrpErrors      = []error{}
	totalVRRPErrors = 0.0

	vrrpStates = []string{vrrpStatusInitialize, vrrpStatusMaster, vrrpStatusBackup}
)

type VrrpVrInfo struct {
	Vrid      int
	Interface string
	V6Info    VrrpInstanceInfo `json:"v6"`
	V4Info    VrrpInstanceInfo `json:"v4"`
}

type VrrpInstanceInfo struct {
	Subinterface string `json:"interface"`
	Status       string
	Statistics   VrrpInstanceStats `json:"stats"`
}

type VrrpInstanceStats struct {
	AdverTx         *int
	AdverRx         *int
	GarpTx          *int
	NeighborAdverTx *int
	Transitions     *int
}

// VRRPCollector collects VRRP metrics, implemented as per prometheus.Collector interface.
type VRRPCollector struct{}

// NewVRRPCollector returns a VRRPCollector struct.
func NewVRRPCollector() *VRRPCollector {
	return &VRRPCollector{}
}

// Name of the collector. Used to populate flag name.
func (*VRRPCollector) Name() string {
	return vrrpSubsystem
}

// Help describes the metrics this collector scrapes. Used to populate flag help.
func (*VRRPCollector) Help() string {
	return "Collect VRRP Metrics"
}

// EnabledByDefault describes whether this collector is enabled by default. Used to populate flag default.
func (*VRRPCollector) EnabledByDefault() bool {
	return false
}

// Describe implemented as per the prometheus.Collector interface.
func (*VRRPCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range getVRRPDesc() {
		ch <- desc
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (c *VRRPCollector) Collect(ch chan<- prometheus.Metric) {
	collectVRRP(ch)
}

// CollectErrors returns what errors have been gathered.
func (*VRRPCollector) CollectErrors() []error {
	return vrrpErrors
}

// CollectTotalErrors returns total errors.
func (*VRRPCollector) CollectTotalErrors() float64 {
	return totalVRRPErrors
}

func getVRRPDesc() map[string]*prometheus.Desc {
	if vrrpDesc != nil {
		return vrrpDesc
	}

	vrrpLabels := []string{"proto", "vrid", "interface", "subinterface"}
	vrrpStateLabels := append(vrrpLabels, "state")

	vrrpDesc = map[string]*prometheus.Desc{
		"vrrpState":       colPromDesc(vrrpSubsystem, "state", "Status of the VRRP state machine.", vrrpStateLabels),
		"adverTx":         colPromDesc(vrrpSubsystem, "adverTx_total", "Advertisements sent total.", vrrpLabels),
		"adverRx":         colPromDesc(vrrpSubsystem, "adverRx_total", "Advertisements received total.", vrrpLabels),
		"garpTx":          colPromDesc(vrrpSubsystem, "garpTx_total", "Gratuitous ARP sent total.", vrrpLabels),
		"neighborAdverTx": colPromDesc(vrrpSubsystem, "neighborAdverTx_total", "Neighbor Advertisements sent total.", vrrpLabels),
		"transitions":     colPromDesc(vrrpSubsystem, "state_transitions_total", "Number of transitions of the VRRP state machine in total.", vrrpLabels),
	}

	return vrrpDesc
}

func collectVRRP(ch chan<- prometheus.Metric) {
	vrrpErrors = []error{}
	jsonVRRPInfo, err := getVRRPInfo()
	if err != nil {
		totalVRRPErrors++
		vrrpErrors = append(vrrpErrors, fmt.Errorf("cannot get vrrp info: %s", err))
	} else {
		if err := processVRRPInfo(ch, jsonVRRPInfo); err != nil {
			totalVRRPErrors++
			vrrpErrors = append(vrrpErrors, err)
		}
	}
}

func getVRRPInfo() ([]byte, error) {
	args := []string{"-c", "show vrrp json"}

	return execVtyshCommand(args...)
}

func processVRRPInfo(ch chan<- prometheus.Metric, jsonVRRPInfo []byte) error {
	var jsonList []VrrpVrInfo
	if err := json.Unmarshal(jsonVRRPInfo, &jsonList); err != nil {
		return fmt.Errorf("cannot unmarshal vrrp json: %s", err)
	}

	for _, vrInfo := range jsonList {
		processInstance(ch, "v4", vrInfo.Vrid, vrInfo.Interface, vrInfo.V4Info)
		processInstance(ch, "v6", vrInfo.Vrid, vrInfo.Interface, vrInfo.V6Info)
	}

	return nil
}

func processInstance(ch chan<- prometheus.Metric, proto string, vrid int, iface string, instance VrrpInstanceInfo) {
	vrrpDesc := getVRRPDesc()

	vrrpLabels := []string{proto, strconv.Itoa(vrid), iface, instance.Subinterface}

	for _, state := range vrrpStates {
		stateLabels := append(vrrpLabels, state)

		var value float64

		if strings.EqualFold(instance.Status, state) {
			value = 1
		}

		newGauge(ch, vrrpDesc["vrrpState"], value, stateLabels...)
	}

	if instance.Statistics.AdverTx != nil {
		newCounter(ch, vrrpDesc["adverTx"], float64(*instance.Statistics.AdverTx), vrrpLabels...)
	}

	if instance.Statistics.AdverRx != nil {
		newCounter(ch, vrrpDesc["adverRx"], float64(*instance.Statistics.AdverRx), vrrpLabels...)
	}

	if instance.Statistics.GarpTx != nil {
		newCounter(ch, vrrpDesc["garpTx"], float64(*instance.Statistics.GarpTx), vrrpLabels...)
	}

	if instance.Statistics.NeighborAdverTx != nil {
		newCounter(ch, vrrpDesc["neighborAdverTx"], float64(*instance.Statistics.NeighborAdverTx), vrrpLabels...)
	}

	if instance.Statistics.Transitions != nil {
		newCounter(ch, vrrpDesc["transitions"], float64(*instance.Statistics.Transitions), vrrpLabels...)
	}
}
