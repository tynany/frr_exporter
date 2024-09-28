package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var pimSubsystem = "pim"

func init() {
	registerCollector(pimSubsystem, disabledByDefault, NewPIMCollector)
}

type pimCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewPIMCollector collects PIM metrics, implemented as per the Collector interface.
func NewPIMCollector(logger *slog.Logger) (Collector, error) {
	return &pimCollector{logger: logger, descriptions: getPIMDesc()}, nil
}

func getPIMDesc() map[string]*prometheus.Desc {
	labels := []string{"vrf"}
	neighborLabels := append(labels, "iface", "neighbor")

	return map[string]*prometheus.Desc{
		"neighborCount": colPromDesc(pimSubsystem, "neighbor_count_total", "Number of neighbors detected", labels),
		"upTime":        colPromDesc(pimSubsystem, "neighbor_uptime_seconds", "How long has the peer been up.", neighborLabels),
	}
}

// Collect implemented as per the Collector interface
func (c *pimCollector) Update(ch chan<- prometheus.Metric) error {
	cmd := "show ip pim vrf all neighbor json"
	jsonPIMNeighbors, err := executePIMCommand(cmd)
	if err != nil {
		return err
	}
	if err := processPIMNeighbors(ch, jsonPIMNeighbors, c.logger, c.descriptions); err != nil {
		return cmdOutputProcessError(cmd, string(jsonPIMNeighbors), err)
	}
	return nil
}

func processPIMNeighbors(ch chan<- prometheus.Metric, jsonPIMNeighbors []byte, logger *slog.Logger, pimDesc map[string]*prometheus.Desc) error {
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(jsonPIMNeighbors, &jsonMap); err != nil {
		return fmt.Errorf("cannot unmarshal pim neighbors json: %s", err)
	}
	for vrfName, vrfData := range jsonMap {
		neighborCount := 0.0
		var _tempvrfInstance map[string]json.RawMessage
		if err := json.Unmarshal(vrfData, &_tempvrfInstance); err != nil {
			return fmt.Errorf("cannot unmarshal VRF instance json: %s", err)
		}
		for ifaceName, ifaceData := range _tempvrfInstance {
			var neighbors map[string]pimNeighbor
			if err := json.Unmarshal(ifaceData, &neighbors); err != nil {
				return fmt.Errorf("cannot unmarshal neighbor json: %s", err)
			}
			for neighborIP, neighborData := range neighbors {
				neighborCount++
				if uptimeSec, err := parseHMS(neighborData.UpTime); err != nil {
					logger.Error("cannot parse neighbor uptime", "uptime", neighborData.UpTime, "err", err)
				} else {
					// The labels are "vrf", "iface", "neighbor"
					neighborLabels := []string{strings.ToLower(vrfName), strings.ToLower(ifaceName), neighborIP}
					newGauge(ch, pimDesc["upTime"], float64(uptimeSec), neighborLabels...)
				}

			}
		}
		newGauge(ch, pimDesc["neighborCount"], neighborCount, vrfName)
	}
	return nil
}

func parseHMS(st string) (uint64, error) {
	var h, m, s uint64
	n, err := fmt.Sscanf(st, "%d:%d:%d", &h, &m, &s)
	if err != nil || n != 3 {
		return 0, err
	}
	return h*3600 + m*60 + s, nil
}

type pimNeighbor struct {
	Interface string
	Neighbor  string
	UpTime    string
}
