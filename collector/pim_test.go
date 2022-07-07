package collector

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var (
	pimNeighborOutput = []byte(`{  "red": {
		"red":{
		},
		"eth2":{
		  "192.0.2.227":{
			"interface":"eth2",
			"neighbor":"192.0.2.227",
			"upTime":"03:45:43",
			"holdTime":"00:01:43",
			"holdTimeMax":105,
			"drPriority":1
		  }
		}
	  }
	  ,  "blue": {
		"blue":{
		},
		"eth1":{
		  "192.0.2.45":{
			"interface":"eth1",
			"neighbor":"192.0.2.45",
			"upTime":"03:45:45",
			"holdTime":"00:01:34",
			"holdTimeMax":105,
			"drPriority":1
		  }
		}
	  }
	  ,  "default": {
		"eth0":{
		  "192.0.2.99":{
			"interface":"eth1",
			"neighbor":"192.0.2.99",
			"upTime":"00:45:45",
			"holdTime":"00:02:34",
			"holdTimeMax":105,
			"drPriority":1
		  }
		}
	  }
	}`)
	expectedPIMMetrics = map[string]float64{
		"frr_pim_neighbor_uptime_seconds{iface=eth2,neighbor=192.0.2.227,vrf=red}":    13543,
		"frr_pim_neighbor_uptime_seconds{iface=eth1,neighbor=192.0.2.45,vrf=blue}":    13545,
		"frr_pim_neighbor_uptime_seconds{iface=eth0,neighbor=192.0.2.99,vrf=default}": 2745,
		"frr_pim_neighbor_count_total{vrf=red}":                                       1,
		"frr_pim_neighbor_count_total{vrf=blue}":                                      1,
		"frr_pim_neighbor_count_total{vrf=default}":                                   1,
	}
	parseHMStests = []struct {
		in  string
		out uint64
	}{
		{"03:45:43", 13543},
		{"00:04:01", 241},
		{"10:00:43", 36043},
	}
)

func TestProcessPIMNeighbors(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processPIMNeighbors(ch, pimNeighborOutput, nil, getPIMDesc()); err != nil {
		t.Errorf("error calling processPIMNeighbors: %s", err)
	}
	close(ch)

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
		if expectedMetricVal, ok := expectedPIMMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedPIMMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}

func TestParseHMS(t *testing.T) {
	for _, tt := range parseHMStests {
		t.Run(tt.in, func(t *testing.T) {
			if uptimeSec, err := parseHMS(tt.in); err != nil || uptimeSec != tt.out {
				t.Errorf("ParseHMS => %s, got %d, wanted %d (err %s)", tt.in, uptimeSec, tt.out, err)
			}
		})
	}
}
