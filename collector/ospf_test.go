package collector

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

func collectMetrics(t *testing.T, ch <-chan prometheus.Metric) map[string]float64 {
	got := make(map[string]float64)
	re := regexp.MustCompile(`.*fqName: "(.*)", help:.*`)

	for m := range ch {
		var dtoM dto.Metric
		if err := m.Write(&dtoM); err != nil {
			t.Errorf("Write(): %v", err)
			continue
		}

		// build label strings WITHOUT quotes
		var lbls []string
		for _, l := range dtoM.GetLabel() {
			lbls = append(lbls, fmt.Sprintf("%s=%s", l.GetName(), l.GetValue()))
		}
		// sort them so the order is deterministic: area,iface,instance,vrf
		sort.Strings(lbls)

		// grab the numeric value
		var v float64
		if c := dtoM.GetCounter(); c != nil {
			v = c.GetValue()
		} else if g := dtoM.GetGauge(); g != nil {
			v = g.GetValue()
		}

		// extract the metric name from the Desc() text
		name := re.FindStringSubmatch(m.Desc().String())[1]
		key := fmt.Sprintf("%s{%s}", name, strings.Join(lbls, ","))
		got[key] = v
	}
	return got
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
