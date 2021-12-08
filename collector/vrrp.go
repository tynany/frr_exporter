package collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	vrrpStatusInitialize = "Initialize"
	vrrpStatusBackup     = "Backup"
	vrrpStatusMaster     = "Master"
)

var (
	vrrpSubsystem = "vrrp"

	vrrpStates = []string{vrrpStatusInitialize, vrrpStatusMaster, vrrpStatusBackup}
)

func init() {
	registerCollector(vrrpSubsystem, disabledByDefault, NewVRRPCollector)
}

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

type vrrpCollector struct {
	logger       log.Logger
	descriptions map[string]*prometheus.Desc
}

// NewVRRPCollector collects VRRP metrics, implemented as per the Collector interface.
func NewVRRPCollector(logger log.Logger) (Collector, error) {
	return &vrrpCollector{logger: logger, descriptions: getVRRPDesc()}, nil
}

func getVRRPDesc() map[string]*prometheus.Desc {
	labels := []string{"proto", "vrid", "interface", "subinterface"}
	stateLabels := append(labels, "state")

	return map[string]*prometheus.Desc{
		"vrrpState":       colPromDesc(vrrpSubsystem, "state", "Status of the VRRP state machine.", stateLabels),
		"adverTx":         colPromDesc(vrrpSubsystem, "adverTx_total", "Advertisements sent total.", labels),
		"adverRx":         colPromDesc(vrrpSubsystem, "adverRx_total", "Advertisements received total.", labels),
		"garpTx":          colPromDesc(vrrpSubsystem, "garpTx_total", "Gratuitous ARP sent total.", labels),
		"neighborAdverTx": colPromDesc(vrrpSubsystem, "neighborAdverTx_total", "Neighbor Advertisements sent total.", labels),
		"transitions":     colPromDesc(vrrpSubsystem, "state_transitions_total", "Number of transitions of the VRRP state machine in total.", labels),
	}
}

// Update implemented as per the Collector interface.
func (c *vrrpCollector) Update(ch chan<- prometheus.Metric) error {
	jsonVRRPInfo, err := getVRRPInfo()
	if err != nil {
		return fmt.Errorf("cannot get vrrp info: %w", err)
	} else {
		if err := processVRRPInfo(ch, jsonVRRPInfo, c.descriptions); err != nil {
			return err
		}
	}
	return nil
}

func getVRRPInfo() ([]byte, error) {
	args := []string{"-c", "show vrrp json"}

	return execVtyshCommand(args...)
}

func processVRRPInfo(ch chan<- prometheus.Metric, jsonVRRPInfo []byte, desc map[string]*prometheus.Desc) error {
	var jsonList []VrrpVrInfo
	if err := json.Unmarshal(jsonVRRPInfo, &jsonList); err != nil {
		return fmt.Errorf("cannot unmarshal vrrp json: %s", err)
	}

	for _, vrInfo := range jsonList {
		processInstance(ch, "v4", vrInfo.Vrid, vrInfo.Interface, vrInfo.V4Info, desc)
		processInstance(ch, "v6", vrInfo.Vrid, vrInfo.Interface, vrInfo.V6Info, desc)
	}

	return nil
}

func processInstance(ch chan<- prometheus.Metric, proto string, vrid int, iface string, instance VrrpInstanceInfo, vrrpDesc map[string]*prometheus.Desc) {

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
