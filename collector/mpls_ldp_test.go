package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestProcessMPLSLDPBindings(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processBindings(ch, readTestFixture(t, "show_mpls_ldp_binding.json"), getMPLSLDPDesc()); err != nil {
		t.Fatalf("error calling processBindings: %s", err)
	}
	close(ch)

	expected := map[string]float64{
		"frr_mpls_ldp_binding_count{address_family=ipv4}": 5,
	}
	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}

func TestProcessMPLSLDPIGPSync(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processIGPSync(ch, readTestFixture(t, "show_mpls_ldp_igp_sync.json"), getMPLSLDPDesc()); err != nil {
		t.Fatalf("error calling processIGPSync: %s", err)
	}
	close(ch)

	expected := map[string]float64{
		"frr_mpls_ldp_igp_sync_state{interface=eth0,peer_ldp_id=}": 0,
	}
	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}

func TestProcessMPLSLDPInterface(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processInterface(ch, readTestFixture(t, "show_mpls_ldp_interface.json"), getMPLSLDPDesc()); err != nil {
		t.Fatalf("error calling processInterface: %s", err)
	}
	close(ch)

	expected := map[string]float64{
		"frr_mpls_ldp_interface_state{address_family=ipv4,name=eth0}":                  1,
		"frr_mpls_ldp_interface_hello_interval_seconds{address_family=ipv4,name=eth0}": 5,
		"frr_mpls_ldp_interface_hello_holdtime_seconds{address_family=ipv4,name=eth0}": 15,
		"frr_mpls_ldp_interface_adjacency_count{address_family=ipv4,name=eth0}":        0,
	}
	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}

func TestProcessMPLSLDPNeighbor(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processNeighbor(ch, readTestFixture(t, "show_mpls_ldp_neighbor.json"), getMPLSLDPDesc()); err != nil {
		t.Fatalf("error calling processNeighbor: %s", err)
	}
	close(ch)

	expected := map[string]float64{
		"frr_mpls_ldp_neighbor_state{address_family=ipv4,neighbor_id=1.1.1.1}":          1,
		"frr_mpls_ldp_neighbor_uptime_seconds{address_family=ipv4,neighbor_id=1.1.1.1}": 141, // 2m21s = 141s
	}
	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}

func TestProcessMPLSLDPDiscovery(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processDiscovery(ch, readTestFixture(t, "show_mpls_ldp_discovery.json"), getMPLSLDPDesc()); err != nil {
		t.Fatalf("error calling processDiscovery: %s", err)
	}
	close(ch)

	expected := map[string]float64{
		"frr_mpls_ldp_discovery_adjacency_count{address_family=ipv4,interface=eth0,neighbor_id=1.1.1.1,type=link}": 1,
	}
	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}
