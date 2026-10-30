package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var expectedEVPNMetrics = map[string]float64{
	"frr_evpn_vni_remote_vtep{flood=HER,remote_vtep=192.0.2.1,vni=4096}": 1,
	"frr_evpn_vni_remote_vtep{flood=HER,remote_vtep=192.0.2.1,vni=4097}": 1,
	"frr_evpn_vni_remote_vtep{flood=HER,remote_vtep=192.0.2.3,vni=4099}": 1,
	"frr_evpn_vni_remote_vtep{flood=-,remote_vtep=192.0.2.1,vni=4099}":   1,
}

func TestProcessEVPNVNIDetail(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processEVPNVNIDetail(ch, readTestFixture(t, "show_evpn_vni_detail.json"), getEVPNDesc()); err != nil {
		t.Errorf("error calling processEVPNVNIDetail: %s", err)
	}
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedEVPNMetrics)
}

func TestProcessEVPNVNIDetailNoEVPN(t *testing.T) {
	for _, output := range []string{"", "{}\n"} {
		ch := make(chan prometheus.Metric, 1024)
		if err := processEVPNVNIDetail(ch, []byte(output), getEVPNDesc()); err != nil {
			t.Errorf("error calling processEVPNVNIDetail with %q: %s", output, err)
		}
		close(ch)

		gotMetrics := collectMetrics(t, ch)
		compareMetrics(t, gotMetrics, map[string]float64{})
	}
}
