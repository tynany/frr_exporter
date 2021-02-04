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
	bfdPeers = []byte(`[
		{
			"multihop": false,
			"peer": "10.10.141.61",
			"local": "10.10.141.81",
			"vrf": "default",
			"id": 869087474,
			"remote-id": 533345668,
			"status": "up",
			"uptime": 847716,
			"diagnostic": "ok",
			"remote-diagnostic": "ok",
			"receive-interval": 300,
			"transmit-interval": 300,
			"echo-interval": 0,
			"remote-receive-interval": 300,
			"remote-transmit-interval": 300,
			"remote-echo-interval": 300
		},
		{
			"multihop": false,
			"peer": "10.10.141.62",
			"local": "10.10.141.81",
			"vrf": "default",
			"id": 2809641312,
			"remote-id": 3617154307,
			"status": "up",
			"uptime": 847595,
			"diagnostic": "ok",
			"remote-diagnostic": "ok",
			"receive-interval": 300,
			"transmit-interval": 300,
			"echo-interval": 0,
			"remote-receive-interval": 300,
			"remote-transmit-interval": 300,
			"remote-echo-interval": 300
		},
		{
			"multihop": false,
			"peer": "10.10.141.63",
			"local": "10.10.141.81",
			"vrf": "default",
			"id": 2809641312,
			"remote-id": 3617154307,
			"status": "down",
			"uptime": 847888,
			"diagnostic": "ok",
			"remote-diagnostic": "ok",
			"receive-interval": 300,
			"transmit-interval": 300,
			"echo-interval": 0,
			"remote-receive-interval": 300,
			"remote-transmit-interval": 300,
			"remote-echo-interval": 300
		}
	]
`)
	expectedBFDMetrics = map[string]float64{
		"frr_bfd_peer_count{}": 3,
		"frr_bfd_peer_uptime{local=10.10.141.81,peer=10.10.141.61}": 847716,
		"frr_bfd_peer_state{local=10.10.141.81,peer=10.10.141.61}":  1,
		"frr_bfd_peer_uptime{local=10.10.141.81,peer=10.10.141.62}": 847595,
		"frr_bfd_peer_state{local=10.10.141.81,peer=10.10.141.62}":  1,
		"frr_bfd_peer_uptime{local=10.10.141.81,peer=10.10.141.63}": 847888,
		"frr_bfd_peer_state{local=10.10.141.81,peer=10.10.141.63}":  0,
	}
)

func TestProcessBFDPeers(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processBFDPeers(ch, bfdPeers); err != nil {
		t.Errorf("error calling processBFDPeers ipv4unicast: %s", err)
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
		if expectedMetricVal, ok := expectedBFDMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedBFDMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}
