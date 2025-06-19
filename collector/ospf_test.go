package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// runOSPFTest is a one-stop helper.
//   - fixture: filename under testdata/
//   - processFn: e.g. processOSPFInterface
//   - getDesc:   e.g. getOSPFIfaceDesc
//   - expected:  map[string]float64
func runOSPFTest(
	t *testing.T,
	fixture string,
	processFn func(chan<- prometheus.Metric, []byte, map[string]*prometheus.Desc, int) error,
	getDesc func() map[string]*prometheus.Desc,
	expected map[string]float64,
) {
	// load the raw JSON
	data := readTestFixture(t, fixture)

	// enough buffer for instance=0 plus instances 1,2
	ch := make(chan prometheus.Metric, len(expected)*3)

	*frrOSPFInstances = ""
	if err := processFn(ch, data, getDesc(), 0); err != nil {
		t.Errorf("instance=0: %v", err)
	}

	*frrOSPFInstances = "1,2"
	for i := 1; i <= 2; i++ {
		if err := processFn(ch, data, getDesc(), i); err != nil {
			t.Errorf("instance=%d: %v", i, err)
		}
	}
	close(ch)

	got := collectMetrics(t, ch)
	compareMetrics(t, got, expected)
}

func TestProcessOSPFInterface(t *testing.T) {
	expected := map[string]float64{
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp1,vrf=default}":                       0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp2,vrf=default}":                       1,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp3,vrf=red}":                           0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp4,vrf=red}":                           1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp1,vrf=default}":            0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp2,vrf=default}":            1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp3,vrf=red}":                0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp4,vrf=red}":                1,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp1,instance=1,vrf=default}":            0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp2,instance=1,vrf=default}":            1,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp3,instance=1,vrf=red}":                0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp4,instance=1,vrf=red}":                1,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp1,instance=2,vrf=default}":            0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp2,instance=2,vrf=default}":            1,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp3,instance=2,vrf=red}":                0,
		"frr_ospf_neighbors{area=0.0.0.0,iface=swp4,instance=2,vrf=red}":                1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp1,instance=1,vrf=default}": 0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp2,instance=1,vrf=default}": 1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp3,instance=1,vrf=red}":     0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp4,instance=1,vrf=red}":     1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp1,instance=2,vrf=default}": 0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp2,instance=2,vrf=default}": 1,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp3,instance=2,vrf=red}":     0,
		"frr_ospf_neighbor_adjacencies{area=0.0.0.0,iface=swp4,instance=2,vrf=red}":     1,
	}
	runOSPFTest(
		t,
		"show_ip_ospf_vrf_all_interface.json",
		processOSPFInterface,
		getOSPFIfaceDesc,
		expected,
	)
}

func TestProcessOSPF(t *testing.T) {
	expected := map[string]float64{
		"frr_ospf_lsa_external_counter{vrf=default}":                            109,
		"frr_ospf_lsa_as_opaque_counter{vrf=default}":                           0,
		"frr_ospf_area_lsa_number{area=0.0.0.0,vrf=default}":                    17,
		"frr_ospf_area_lsa_network_number{area=0.0.0.0,vrf=default}":            1,
		"frr_ospf_area_lsa_summary_number{area=0.0.0.0,vrf=default}":            0,
		"frr_ospf_area_lsa_asbr_number{area=0.0.0.0,vrf=default}":               0,
		"frr_ospf_area_lsa_nssa_number{area=0.0.0.0,vrf=default}":               0,
		"frr_ospf_lsa_external_counter{instance=1,vrf=default}":                 109,
		"frr_ospf_lsa_as_opaque_counter{instance=1,vrf=default}":                0,
		"frr_ospf_area_lsa_number{area=0.0.0.0,instance=1,vrf=default}":         17,
		"frr_ospf_area_lsa_network_number{area=0.0.0.0,instance=1,vrf=default}": 1,
		"frr_ospf_area_lsa_summary_number{area=0.0.0.0,instance=1,vrf=default}": 0,
		"frr_ospf_area_lsa_asbr_number{area=0.0.0.0,instance=1,vrf=default}":    0,
		"frr_ospf_area_lsa_nssa_number{area=0.0.0.0,instance=1,vrf=default}":    0,
		"frr_ospf_lsa_external_counter{instance=2,vrf=default}":                 109,
		"frr_ospf_lsa_as_opaque_counter{instance=2,vrf=default}":                0,
		"frr_ospf_area_lsa_number{area=0.0.0.0,instance=2,vrf=default}":         17,
		"frr_ospf_area_lsa_network_number{area=0.0.0.0,instance=2,vrf=default}": 1,
		"frr_ospf_area_lsa_summary_number{area=0.0.0.0,instance=2,vrf=default}": 0,
		"frr_ospf_area_lsa_asbr_number{area=0.0.0.0,instance=2,vrf=default}":    0,
		"frr_ospf_area_lsa_nssa_number{area=0.0.0.0,instance=2,vrf=default}":    0,
	}
	runOSPFTest(
		t,
		"show_ip_ospf_vrf_all.json",
		processOSPF,
		getOSPFDesc,
		expected,
	)
}

func TestProcessOSPFNeigh(t *testing.T) {
	expected := map[string]float64{
		"frr_ospf_neighbor_state{iface=eth1,instance=1,local_address=192.168.4.2,neighbor=0.0.32.237,remote_address=192.168.4.3,vrf=default}": 4,
		"frr_ospf_neighbor_state{iface=eth0,instance=2,local_address=192.168.1.2,neighbor=0.0.35.148,remote_address=192.168.1.3,vrf=default}": 7,
		"frr_ospf_neighbor_state{iface=eth1,instance=2,local_address=192.168.2.2,neighbor=0.0.35.148,remote_address=192.168.2.3,vrf=default}": 4,
		"frr_ospf_neighbor_state{iface=eth0,instance=2,local_address=192.168.3.2,neighbor=0.0.32.237,remote_address=192.168.3.3,vrf=default}": 6,
		"frr_ospf_neighbor_state{iface=eth0,local_address=192.168.3.2,neighbor=0.0.32.237,remote_address=192.168.3.3,vrf=default}":            6,
		"frr_ospf_neighbor_state{iface=eth1,local_address=192.168.4.2,neighbor=0.0.32.237,remote_address=192.168.4.3,vrf=default}":            4,
		"frr_ospf_neighbor_state{iface=eth0,instance=1,local_address=192.168.1.2,neighbor=0.0.35.148,remote_address=192.168.1.3,vrf=default}": 7,
		"frr_ospf_neighbor_state{iface=eth0,instance=1,local_address=192.168.3.2,neighbor=0.0.32.237,remote_address=192.168.3.3,vrf=default}": 6,
		"frr_ospf_neighbor_state{iface=eth0,local_address=192.168.1.2,neighbor=0.0.35.148,remote_address=192.168.1.3,vrf=default}":            7,
		"frr_ospf_neighbor_state{iface=eth1,local_address=192.168.2.2,neighbor=0.0.35.148,remote_address=192.168.2.3,vrf=default}":            4,
		"frr_ospf_neighbor_state{iface=eth1,instance=1,local_address=192.168.2.2,neighbor=0.0.35.148,remote_address=192.168.2.3,vrf=default}": 4,
		"frr_ospf_neighbor_state{iface=eth1,instance=2,local_address=192.168.4.2,neighbor=0.0.32.237,remote_address=192.168.4.3,vrf=default}": 4,
	}
	runOSPFTest(
		t,
		"show_ip_ospf_vrf_all_neighbors.json",
		processOSPFNeigh,
		getOSPFNeighDesc,
		expected,
	)
}

func TestProcessOSPFDataMaxAge(t *testing.T) {
	expected := map[string]float64{
		"frr_ospf_data_ls_max_age{instance=default,vrf=2}": 2,
		"frr_ospf_data_ls_max_age{vrf=default}":            2,
		"frr_ospf_data_ls_max_age{instance=default,vrf=1}": 2,
	}
	runOSPFTest(
		t,
		"show_ip_ospf_vrf_all_database_max_age.json",
		processOSPFDataMaxAge,
		getOSPFDataMaxAgeDesc,
		expected,
	)
}
