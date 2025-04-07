package collector

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

func TestProcessRouteSummaries(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)

	enableDetailedRoutes := true
	detailedRoutes = &enableDetailedRoutes

	// Load test data for IPv4
	jsonRouteIPv4 := readTestFixture(t, "show_ip_route_vrf_all_summary.json")
	if err := processRouteSummaries(ch, jsonRouteIPv4, "ipv4", getRouteDesc()); err != nil {
		t.Errorf("error calling processRouteSummaries ipv4: %s", err)
	}

	// Load test data for IPv6
	jsonRouteIPv6 := readTestFixture(t, "show_ipv6_route_vrf_all_summary.json")
	if err := processRouteSummaries(ch, jsonRouteIPv6, "ipv6", getRouteDesc()); err != nil {
		t.Errorf("error calling processRouteSummaries ipv6: %s", err)
	}

	close(ch)

	// Create a map of following format:
	//   key: metric_name{labelname:labelvalue,...}
	//   value: metric value
	gotMetrics := make(map[string]float64)

	for {
		msg, more := <-ch
		if !more {
			break
		}
		metric := &dto.Metric{}
		if err := msg.Write(metric); err != nil {
			t.Errorf("error writing metric: %s", err)
		}

		var labels []string
		for _, label := range metric.GetLabel() {
			labels = append(labels, fmt.Sprintf("%s=%s", label.GetName(), label.GetValue()))
		}

		var value float64
		if metric.GetCounter() != nil {
			value = metric.GetCounter().GetValue()
		} else if metric.GetGauge() != nil {
			value = metric.GetGauge().GetValue()
		}

		re, err := regexp.Compile(`.*fqName: "(.*)", help:.*`)
		if err != nil {
			t.Errorf("could not compile regex: %s", err)
		}
		metricName := re.FindStringSubmatch(msg.Desc().String())[1]

		gotMetrics[fmt.Sprintf("%s{%s}", metricName, strings.Join(labels, ","))] = value
	}

	for metricName, metricVal := range gotMetrics {
		if expectedMetricVal, ok := expectedRouteMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}
		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedRouteMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}
