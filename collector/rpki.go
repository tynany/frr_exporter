package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var rpkiSubsystem = "rpki"

func init() {
	registerCollector(rpkiSubsystem, disabledByDefault, NewRPKICollector)
}

type rpkiCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewRPKICollector collects RPKI cache-connection metrics, implemented as per the Collector interface.
func NewRPKICollector(logger *slog.Logger) (Collector, error) {
	return &rpkiCollector{logger: logger, descriptions: getRPKIDesc()}, nil
}

func getRPKIDesc() map[string]*prometheus.Desc {
	labels := []string{"vrf", "mode", "host", "port"}
	return map[string]*prometheus.Desc{
		"cacheState":      colPromDesc(rpkiSubsystem, "cache_state", "State of the RPKI cache connection (1 = connected, 0 = disconnected).", labels),
		"cachePreference": colPromDesc(rpkiSubsystem, "cache_preference", "Preference value of the RPKI cache connection.", labels),
	}
}

// Update implemented as per the Collector interface.
func (c *rpkiCollector) Update(ch chan<- prometheus.Metric) error {
	vrfs, err := getVRFs()
	if err != nil {
		return err
	}

	for _, vrf := range vrfs {
		var cmd string
		if vrf == "default" {
			cmd = "show rpki cache-connection json"
		} else {
			cmd = fmt.Sprintf("show rpki cache-connection vrf %s json", vrf)
		}

		output, err := executeBGPCommand(cmd)
		if err != nil {
			return err
		}
		if len(output) == 0 {
			continue
		}

		if err := processRPKICacheConnection(ch, output, vrf, c.descriptions); err != nil {
			return cmdOutputProcessError(cmd, string(output), err)
		}
	}
	return nil
}

func processRPKICacheConnection(ch chan<- prometheus.Metric, jsonRPKI []byte, vrf string, rpkiDesc map[string]*prometheus.Desc) error {
	var cacheConn rpkiCacheConnection
	if err := json.Unmarshal(jsonRPKI, &cacheConn); err != nil {
		return err
	}

	for _, conn := range cacheConn.Connections {
		labels := []string{vrf, conn.Mode, conn.Host, strconv.Itoa(conn.Port)}

		state := 0.0
		if conn.State == "connected" {
			state = 1.0
		}

		newGauge(ch, rpkiDesc["cacheState"], state, labels...)
		newGauge(ch, rpkiDesc["cachePreference"], float64(conn.Preference), labels...)
	}
	return nil
}

type rpkiCacheConnection struct {
	ConnectedGroup int              `json:"connectedGroup"`
	Connections    []rpkiConnection `json:"connections"`
}

type rpkiConnection struct {
	Mode       string `json:"mode"`
	Host       string `json:"host"`
	Port       int    `json:"port,string"`
	Preference int    `json:"preference"`
	State      string `json:"state"`
}
