package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	routeSubsystem = "route"
	detailedRoutes = kingpin.Flag("collector.route.detailed-routes", "Enable detailed route count of each route type (default: disabled).").Default("False").Bool()
)

func init() {
	registerCollector(routeSubsystem, enabledByDefault, NewRouteCollector)
}

type routeCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewRouteCollector collects route summary, implemented as per the Collector interface.
func NewRouteCollector(logger *slog.Logger) (Collector, error) {
	return &routeCollector{logger: logger, descriptions: getRouteDesc()}, nil
}

func getRouteDesc() map[string]*prometheus.Desc {
	labels := []string{"afi", "route_type", "vrf"}
	totalLabels := []string{"afi", "vrf"}

	return map[string]*prometheus.Desc{
		"total":             colPromDesc(routeSubsystem, "total", "Total number of routes", totalLabels),
		"totalFib":          colPromDesc(routeSubsystem, "total_fib", "Total number of routes in FIB", totalLabels),
		"fibCount":          colPromDesc(routeSubsystem, "fib_count", "Number of routes of route type in FIB", labels),
		"fibOffloadedCount": colPromDesc(routeSubsystem, "fib_offloaded_count", "Number of offloaded routes of route type in FIB", labels),
		"fibTrappedCount":   colPromDesc(routeSubsystem, "fib_trapped_count", "Number of trapped routes of route type in FIB", labels),
		"ribCount":          colPromDesc(routeSubsystem, "rib_count", "Number of routes of route type in RIB", labels),
	}
}

// Update implemented as per the Collector interface.
func (c *routeCollector) Update(ch chan<- prometheus.Metric) error {
	cmdIPv4 := "show ip route vrf all summary json"
	cmdIPv6 := "show ipv6 route vrf all summary json"

	jsonRouteIPv4, err := executeZebraCommand(cmdIPv4)
	if err != nil {
		return err
	}

	jsonRouteIPv6, err := executeZebraCommand(cmdIPv6)
	if err != nil {
		return err
	}

	if err := processRouteSummaries(ch, jsonRouteIPv4, "ipv4", c.descriptions); err != nil {
		return cmdOutputProcessError(cmdIPv4, string(jsonRouteIPv4), err)
	}

	if err := processRouteSummaries(ch, jsonRouteIPv6, "ipv6", c.descriptions); err != nil {
		return cmdOutputProcessError(cmdIPv6, string(jsonRouteIPv6), err)
	}
	return nil
}

func processRouteSummaries(ch chan<- prometheus.Metric, jsonRoute []byte, afi string, routeDesc map[string]*prometheus.Desc) error {
	var routeSummaries map[string]routeSummary
	if err := json.Unmarshal(jsonRoute, &routeSummaries); err != nil {
		// fallback for older FRR versions that do not return the VRF key
		var single routeSummary
		if err2 := json.Unmarshal(jsonRoute, &single); err2 != nil {
			// fallback for pre-10.1.0 FRR with multiple VRFs where "vrf all"
			// produces concatenated (invalid) JSON. Query each VRF individually.
			return processRouteSummariesPerVRF(ch, afi, routeDesc)
		}
		routeSummaries = map[string]routeSummary{
			"default": single,
		}
	}

	for vrf, rs := range routeSummaries {
		emitRouteSummaryMetrics(ch, rs, afi, vrf, routeDesc)
	}

	return nil
}

func processRouteSummariesPerVRF(ch chan<- prometheus.Metric, afi string, routeDesc map[string]*prometheus.Desc) error {
	vrfs, err := getVRFs()
	if err != nil {
		return err
	}

	var cmdFmt string
	if afi == "ipv4" {
		cmdFmt = "show ip route vrf %s summary json"
	} else {
		cmdFmt = "show ipv6 route vrf %s summary json"
	}

	for _, vrf := range vrfs {
		cmd := fmt.Sprintf(cmdFmt, vrf)
		jsonRoute, err := executeZebraCommand(cmd)
		if err != nil {
			return err
		}

		var rs routeSummary
		if err := json.Unmarshal(jsonRoute, &rs); err != nil {
			return cmdOutputProcessError(cmd, string(jsonRoute), err)
		}

		emitRouteSummaryMetrics(ch, rs, afi, vrf, routeDesc)
	}

	return nil
}

func emitRouteSummaryMetrics(ch chan<- prometheus.Metric, rs routeSummary, afi string, vrf string, routeDesc map[string]*prometheus.Desc) {
	newGauge(ch, routeDesc["total"], float64(rs.RoutesTotal), afi, vrf)
	newGauge(ch, routeDesc["totalFib"], float64(rs.RoutesTotalFib), afi, vrf)

	if *detailedRoutes {
		for _, route := range rs.Routes {
			labels := []string{afi, route.Type, vrf}
			newGauge(ch, routeDesc["fibCount"], float64(route.Fib), labels...)
			newGauge(ch, routeDesc["fibOffloadedCount"], float64(route.FibOffLoaded), labels...)
			newGauge(ch, routeDesc["fibTrappedCount"], float64(route.FibTrapped), labels...)
			newGauge(ch, routeDesc["ribCount"], float64(route.Rib), labels...)
		}
	}
}

func getVRFs() ([]string, error) {
	output, err := executeZebraCommand("show vrf")
	if err != nil {
		return nil, err
	}
	return parseVRFs(output), nil
}

func parseVRFs(output []byte) []string {
	vrfs := []string{"default"}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "vrf" {
			vrfs = append(vrfs, fields[1])
		}
	}
	return vrfs
}

type routeSummary struct {
	Routes         []route `json:"routes"`
	RoutesTotal    uint32  `json:"routesTotal"`
	RoutesTotalFib uint32  `json:"routesTotalFib"`
}

type route struct {
	Fib          uint32 `json:"fib"`
	Rib          uint32 `json:"rib"`
	FibOffLoaded uint32 `json:"fibOffLoaded"`
	FibTrapped   uint32 `json:"fibTrapped"`
	Type         string `json:"type"`
}
