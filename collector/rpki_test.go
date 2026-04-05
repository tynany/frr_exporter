package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var expectedRPKIMetrics = map[string]float64{
	"frr_rpki_cache_state{host=172.20.15.59,mode=tcp,port=8082,vrf=default}":      1,
	"frr_rpki_cache_preference{host=172.20.15.59,mode=tcp,port=8082,vrf=default}": 10,
	"frr_rpki_cache_state{host=172.20.15.60,mode=tcp,port=8083,vrf=default}":      0,
	"frr_rpki_cache_preference{host=172.20.15.60,mode=tcp,port=8083,vrf=default}": 20,
}

func TestProcessRPKICacheConnection(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processRPKICacheConnection(ch, readTestFixture(t, "show_rpki_cache_connection.json"), "default", getRPKIDesc()); err != nil {
		t.Errorf("error calling processRPKICacheConnection: %s", err)
	}
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedRPKIMetrics)
}

var expectedRPKIVRFMetrics = map[string]float64{
	"frr_rpki_cache_state{host=172.20.15.59,mode=tcp,port=8082,vrf=TEST}":      1,
	"frr_rpki_cache_preference{host=172.20.15.59,mode=tcp,port=8082,vrf=TEST}": 10,
}

func TestProcessRPKICacheConnectionVRF(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processRPKICacheConnection(ch, readTestFixture(t, "show_rpki_cache_connection_vrf_TEST.json"), "TEST", getRPKIDesc()); err != nil {
		t.Errorf("error calling processRPKICacheConnection VRF: %s", err)
	}
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedRPKIVRFMetrics)
}
