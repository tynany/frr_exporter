package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var expectedRouteMetrics = map[string]float64{
	"frr_route_fib_count{afi=ipv4,route_type=connected,vrf=default}":           1,
	"frr_route_fib_count{afi=ipv4,route_type=connected,vrf=red}":               2,
	"frr_route_fib_count{afi=ipv4,route_type=ebgp,vrf=red}":                    1000504,
	"frr_route_fib_count{afi=ipv4,route_type=ibgp,vrf=red}":                    0,
	"frr_route_fib_count{afi=ipv4,route_type=local,vrf=default}":               1,
	"frr_route_fib_count{afi=ipv4,route_type=local,vrf=red}":                   2,
	"frr_route_fib_count{afi=ipv4,route_type=static,vrf=default}":              1,
	"frr_route_fib_count{afi=ipv4,route_type=static,vrf=red}":                  3,
	"frr_route_fib_count{afi=ipv6,route_type=connected,vrf=default}":           2,
	"frr_route_fib_count{afi=ipv6,route_type=connected,vrf=red}":               2,
	"frr_route_fib_count{afi=ipv6,route_type=ebgp,vrf=red}":                    218318,
	"frr_route_fib_count{afi=ipv6,route_type=ibgp,vrf=red}":                    0,
	"frr_route_fib_count{afi=ipv6,route_type=local,vrf=red}":                   1,
	"frr_route_fib_count{afi=ipv6,route_type=static,vrf=red}":                  1,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=connected,vrf=default}": 0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=connected,vrf=red}":     0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=ebgp,vrf=red}":          0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=ibgp,vrf=red}":          0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=local,vrf=default}":     0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=local,vrf=red}":         0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=static,vrf=default}":    0,
	"frr_route_fib_offloaded_count{afi=ipv4,route_type=static,vrf=red}":        0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=connected,vrf=default}": 0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=connected,vrf=red}":     0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=ebgp,vrf=red}":          0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=ibgp,vrf=red}":          0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=local,vrf=red}":         0,
	"frr_route_fib_offloaded_count{afi=ipv6,route_type=static,vrf=red}":        0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=connected,vrf=default}":   0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=connected,vrf=red}":       0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=ebgp,vrf=red}":            0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=ibgp,vrf=red}":            0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=local,vrf=default}":       0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=local,vrf=red}":           0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=static,vrf=default}":      0,
	"frr_route_fib_trapped_count{afi=ipv4,route_type=static,vrf=red}":          0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=connected,vrf=default}":   0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=connected,vrf=red}":       0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=ebgp,vrf=red}":            0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=ibgp,vrf=red}":            0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=local,vrf=red}":           0,
	"frr_route_fib_trapped_count{afi=ipv6,route_type=static,vrf=red}":          0,
	"frr_route_rib_count{afi=ipv4,route_type=connected,vrf=default}":           1,
	"frr_route_rib_count{afi=ipv4,route_type=connected,vrf=red}":               2,
	"frr_route_rib_count{afi=ipv4,route_type=ebgp,vrf=red}":                    1000505,
	"frr_route_rib_count{afi=ipv4,route_type=ibgp,vrf=red}":                    0,
	"frr_route_rib_count{afi=ipv4,route_type=local,vrf=default}":               1,
	"frr_route_rib_count{afi=ipv4,route_type=local,vrf=red}":                   2,
	"frr_route_rib_count{afi=ipv4,route_type=static,vrf=default}":              1,
	"frr_route_rib_count{afi=ipv4,route_type=static,vrf=red}":                  3,
	"frr_route_rib_count{afi=ipv6,route_type=connected,vrf=default}":           2,
	"frr_route_rib_count{afi=ipv6,route_type=connected,vrf=red}":               2,
	"frr_route_rib_count{afi=ipv6,route_type=ebgp,vrf=red}":                    218319,
	"frr_route_rib_count{afi=ipv6,route_type=ibgp,vrf=red}":                    0,
	"frr_route_rib_count{afi=ipv6,route_type=local,vrf=red}":                   1,
	"frr_route_rib_count{afi=ipv6,route_type=static,vrf=red}":                  1,
	"frr_route_total{afi=ipv4,vrf=default}":                                    3,
	"frr_route_total{afi=ipv4,vrf=red}":                                        1000512,
	"frr_route_total{afi=ipv6,vrf=default}":                                    2,
	"frr_route_total{afi=ipv6,vrf=red}":                                        218323,
	"frr_route_total_fib{afi=ipv4,vrf=default}":                                3,
	"frr_route_total_fib{afi=ipv4,vrf=red}":                                    1000511,
	"frr_route_total_fib{afi=ipv6,vrf=default}":                                2,
	"frr_route_total_fib{afi=ipv6,vrf=red}":                                    218322,
}

func TestParseVRFs(t *testing.T) {
	fixture := readTestFixture(t, "show_vrf.txt")
	got := parseVRFs(fixture)
	expected := []string{"default", "vrf-red", "vrf-blue"}

	if len(got) != len(expected) {
		t.Fatalf("expected %d VRFs, got %d: %v", len(expected), len(got), got)
	}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("expected VRF[%d] = %q, got %q", i, v, got[i])
		}
	}
}

func TestParseVRFsEmpty(t *testing.T) {
	fixture := readTestFixture(t, "show_vrf_empty.txt")
	got := parseVRFs(fixture)
	if len(got) != 1 || got[0] != "default" {
		t.Errorf("expected [default], got %v", got)
	}
}

func TestProcessRouteSummaries(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)

	enableDetailedRoutes := true
	detailedRoutes = &enableDetailedRoutes

	jsonRouteIPv4 := readTestFixture(t, "show_ip_route_vrf_all_summary.json")
	if err := processRouteSummaries(ch, jsonRouteIPv4, "ipv4", getRouteDesc()); err != nil {
		t.Fatalf("error calling processRouteSummaries ipv4: %s", err)
	}

	jsonRouteIPv6 := readTestFixture(t, "show_ipv6_route_vrf_all_summary.json")
	if err := processRouteSummaries(ch, jsonRouteIPv6, "ipv6", getRouteDesc()); err != nil {
		t.Fatalf("error calling processRouteSummaries ipv6: %s", err)
	}

	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedRouteMetrics)
}
