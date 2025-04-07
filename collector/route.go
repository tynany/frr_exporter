package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"

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
	logger         *slog.Logger
	descriptions   map[string]*prometheus.Desc
	detailedRoutes bool
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
	cmdIPv4 := fmt.Sprintf("show ip route vrf all summary json")
	cmdIPv6 := fmt.Sprintf("show ipv6 route vrf all summary json")

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
		return err
	}

	for vrf, routeSummary := range routeSummaries {

		// Total routes
		newGauge(ch, routeDesc["total"], float64(routeSummary.RoutesTotal), afi, vrf)

		// Total FIB routes
		newGauge(ch, routeDesc["totalFib"], float64(routeSummary.RoutesTotalFib), afi, vrf)

		if *detailedRoutes {
			for _, route := range routeSummary.Routes {
				labels := []string{afi, route.Type, vrf}

				newGauge(ch, routeDesc["fibCount"], float64(route.Fib), labels...)
				newGauge(ch, routeDesc["fibOffloadedCount"], float64(route.FibOffLoaded), labels...)
				newGauge(ch, routeDesc["fibTrappedCount"], float64(route.FibTrapped), labels...)
				newGauge(ch, routeDesc["ribCount"], float64(route.Rib), labels...)
			}
		}
	}

	return nil
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
