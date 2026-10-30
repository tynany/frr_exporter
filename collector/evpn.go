package collector

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var evpnSubsystem = "evpn"

func init() {
	registerCollector(evpnSubsystem, disabledByDefault, NewEVPNCollector)
}

type evpnCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewEVPNCollector collects EVPN VNI metrics, implemented as per the Collector interface.
func NewEVPNCollector(logger *slog.Logger) (Collector, error) {
	return &evpnCollector{logger: logger, descriptions: getEVPNDesc()}, nil
}

func getEVPNDesc() map[string]*prometheus.Desc {
	remoteVTEPLabels := []string{"vni", "remote_vtep", "flood"}
	return map[string]*prometheus.Desc{
		"remoteVTEP": colPromDesc(evpnSubsystem, "vni_remote_vtep", "Remote VTEP of the VNI and its flood type for BUM traffic (HER = head-end replication, PIM-SM = multicast, - = none). Always 1.", remoteVTEPLabels),
	}
}

// Update implemented as per the Collector interface.
func (c *evpnCollector) Update(ch chan<- prometheus.Metric) error {
	cmd := "show evpn vni detail json"
	jsonEVPNVNIs, err := executeZebraCommand(cmd)
	if err != nil {
		return err
	}
	if err := processEVPNVNIDetail(ch, jsonEVPNVNIs, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(jsonEVPNVNIs), err)
	}
	return nil
}

func processEVPNVNIDetail(ch chan<- prometheus.Metric, jsonEVPNVNIs []byte, evpnDesc map[string]*prometheus.Desc) error {
	// When EVPN is not enabled, FRR prints nothing (before 9.0) or an empty object instead of a list.
	if output := bytes.TrimSpace(jsonEVPNVNIs); len(output) == 0 || bytes.Equal(output, []byte("{}")) {
		return nil
	}

	var vnis []evpnVNI
	if err := json.Unmarshal(jsonEVPNVNIs, &vnis); err != nil {
		return err
	}

	// L3 VNIs are listed too, but have no remote VTEPs.
	for _, vni := range vnis {
		vniLabel := strconv.FormatUint(uint64(vni.VNI), 10)
		for _, vtep := range vni.RemoteVTEPs {
			newGauge(ch, evpnDesc["remoteVTEP"], 1, vniLabel, vtep.IP, vtep.Flood)
		}
	}
	return nil
}

type evpnVNI struct {
	VNI         uint32           `json:"vni"`
	RemoteVTEPs []evpnRemoteVTEP `json:"remoteVteps"`
}

type evpnRemoteVTEP struct {
	IP    string `json:"ip"`
	Flood string `json:"flood"`
}
