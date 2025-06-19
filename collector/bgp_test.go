package collector

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func runBGPSummaryTest(t *testing.T, fixture string, afi string, processFn func(chan<- prometheus.Metric, []byte, string, string, *slog.Logger, map[string]*prometheus.Desc) error, getDesc func() map[string]*prometheus.Desc, expected map[string]float64) {
	// load the raw JSON
	data := readTestFixture(t, fixture)

	// enough buffer for instance=0 plus instances 1,2
	ch := make(chan prometheus.Metric, len(expected)*3)

	if err := processFn(ch, data, afi, "", nil, getDesc()); err != nil {
		t.Errorf("error calling processFn %s: %s", afi, err)
	}
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expected)
}

func TestProcessBGPSummary(t *testing.T) {
	expectedIpv4 := map[string]float64{
		"frr_bgp_peer_groups_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                           0.0,
		"frr_bgp_peer_groups_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                               0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                          0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                              0.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":        100.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":        0.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":            100.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":            200.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":            100.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":            0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                100.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                200.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}": 0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}": 2.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":     2.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":     0.0,
		"frr_bgp_peers_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                 2.0,
		"frr_bgp_peers_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                     2.0,
		"frr_bgp_peers_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                39936.0,
		"frr_bgp_peers_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                    39936.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":                         1.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":                         0.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                             1.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                             0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":                10.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":                0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                    20.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                    0.0,
		"frr_bgp_rib_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                   1.0,
		"frr_bgp_rib_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                       0.0,
		"frr_bgp_rib_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                  64.0,
		"frr_bgp_rib_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                      0.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64512,peer=192.168.0.4,peer_as=64515,safi=unicast,vrf=default}":                         2.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64512,peer=192.168.0.4,peer_as=64515,safi=unicast,vrf=default}":            0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64512,peer=192.168.0.4,peer_as=64515,safi=unicast,vrf=default}": 2.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64512,peer=192.168.0.4,peer_as=64515,safi=unicast,vrf=default}":                0.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64512,peer=192.168.0.4,peer_as=64515,safi=unicast,vrf=default}":        0.0,
	}
	runBGPSummaryTest(t, "show_bgp_vrf_all_ipv4_summary.json", "ipv4", processBGPSummary, getBGPDesc, expectedIpv4)

	expectedIpv6 := map[string]float64{
		"frr_bgp_peers_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                            59904.0,
		"frr_bgp_peers_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                59904.0,
		"frr_bgp_peers_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                             2.0,
		"frr_bgp_peers_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                 2.0,
		"frr_bgp_peer_groups_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                       0.0,
		"frr_bgp_peer_groups_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                           0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                      0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                          0.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":        29285.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":        0.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":          29285.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":          0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":            29285.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":            0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":              29285.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":              0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}": 1.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}": 0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":   1.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":   0.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":                         1.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":                         0.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":                           1.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":                           0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":                8465643000.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":                0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":                  87873.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":                  0.0,
		"frr_bgp_rib_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                               3.0,
		"frr_bgp_rib_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                   3.0,
		"frr_bgp_rib_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                              456.0,
		"frr_bgp_rib_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                  456.0,
	}
	runBGPSummaryTest(t, "show_bgp_vrf_all_ipv6_summary.json", "ipv6", processBGPSummary, getBGPDesc, expectedIpv6)
}

func TestProcessBgpL2vpnEvpnSummary(t *testing.T) {
	expected := map[string]float64{
		"frr_bgp_l2vpn_evpn_arp_nd_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}":      23.000000,
		"frr_bgp_l2vpn_evpn_arp_nd_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}":      0.000000,
		"frr_bgp_l2vpn_evpn_mac_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}":         0.000000,
		"frr_bgp_l2vpn_evpn_mac_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}":         42.000000,
		"frr_bgp_l2vpn_evpn_remote_vtep_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}": -1.000000,
		"frr_bgp_l2vpn_evpn_remote_vtep_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}": 1.000000,
	}

	ch := make(chan prometheus.Metric, 1024)
	if err := processBgpL2vpnEvpnSummary(ch, readTestFixture(t, "show_evpn_vni.json"), getBGPL2VPNDesc()); err != nil {
		t.Errorf("error calling processBgpL2vpnEvpnSummary: %s", err)
	}
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expected)
}

func TestProcessBGPPeerDesc(t *testing.T) {
	expectedOutput := map[string]bgpVRF{
		"default": {
			ID:   0,
			Name: "default",
			BGPNeighbors: map[string]bgpNeighbor{
				"10.1.1.10": {Desc: "{\"desc\":\"rt1\"}"},
				"swp2":      {Desc: "{\"desc\":\"fw1\"}"},
			},
		},
		"vrf1": {
			ID:   -1,
			Name: "vrf1",
			BGPNeighbors: map[string]bgpNeighbor{
				"10.2.0.1": {Desc: "{\"desc\":\"remote\"}"},
			},
		},
	}

	peerDesc, err := processBGPPeerDesc(readTestFixture(t, "show_bgp_vrf_all_neighbors.json"))
	if err != nil {
		t.Errorf("error calling processBGPPeerDesc: %s", err)
	}

	if !reflect.DeepEqual(peerDesc, expectedOutput) {
		t.Errorf("error comparing bgp neighbor description output: %v does not match expected %v", peerDesc, expectedOutput)
	}
}

func TestProcessBGPNexthop(t *testing.T) {
	expected := map[string]bgpNextHop{
		"default": {
			IPv4: map[string]bgpNextHopInterfaces{
				"10.1.2.1": {
					Nexthops: []struct {
						InterfaceName string `json:"interfaceName"`
					}{
						{InterfaceName: "eth1"},
					},
				},
				"10.2.2.1": {
					Nexthops: []struct {
						InterfaceName string `json:"interfaceName"`
					}{
						{InterfaceName: "eth2"},
					},
				},
			},
			IPv6: map[string]bgpNextHopInterfaces{},
		},
	}

	input := readTestFixture(t, "show_ip_bgp_vrf_all_nexthop.json")

	got, err := processBGPNexthop(input)
	if err != nil {
		t.Fatalf("processBGPNexthop returned error: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("processBGPNexthop() =\n%#v\nwant\n%#v", got, expected)
	}
}
