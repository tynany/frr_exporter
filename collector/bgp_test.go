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
	bgpSumV4Unicast = []byte(`{
"default":{
  "routerId":"192.168.0.1",
  "as":64512,
  "vrfId":0,
  "vrfName":"default",
  "tableVersion":0,
  "ribCount":1,
  "ribMemory":64,
  "peerCount":2,
  "peerMemory":39936,
  "peers":{
    "192.168.0.2":{
      "remoteAs":64513,
      "version":4,
      "msgRcvd":100,
      "msgSent":100,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"10000",
      "peerUptimeMsec":10000,
      "prefixReceivedCount":0,
      "state":"Established",
      "idType":"ipv4"
    },
    "192.168.0.3":{
      "remoteAs":64514,
      "version":4,
      "msgRcvd":0,
      "msgSent":0,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"never",
      "peerUptimeMsec":0,
      "prefixReceivedCount":2,
      "state":"Active",
      "idType":"ipv4"
    }
  },
  "totalPeers":2,
  "dynamicPeers":0,
  "bestPath":{
    "multiPathRelax":"false"
  }
}
,
"red":{
  "routerId":"192.168.1.1",
  "as":64612,
  "vrfId":39,
  "vrfName":"red",
  "tableVersion":0,
  "ribCount":0,
  "ribMemory":0,
  "peerCount":2,
  "peerMemory":39936,
  "peers":{
    "192.168.1.2":{
      "remoteAs":64613,
      "version":4,
      "msgRcvd":100,
      "msgSent":100,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"10000",
      "peerUptimeMsec":20000,
      "prefixReceivedCount":2,
      "state":"Established",
      "idType":"ipv4"
    },
    "192.168.1.3":{
      "remoteAs":64614,
      "version":4,
      "msgRcvd":200,
      "msgSent":200,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"never",
      "peerUptimeMsec":0,
      "prefixReceivedCount":0,
      "state":"Active",
      "idType":"ipv4"
    }
  },
  "totalPeers":2,
  "dynamicPeers":0,
  "bestPath":{
    "multiPathRelax":"false"
  }
}
}
`)

	bgpSumV6Unicast = []byte(`{
"default":{
  "routerId":"192.168.0.1",
  "as":64512,
  "vrfId":0,
  "vrfName":"default",
  "tableVersion":6,
  "ribCount":3,
  "ribMemory":456,
  "peerCount":2,
  "peerMemory":59904,
  "peers":{
    "fd00::1":{
      "remoteAs":64513,
      "version":4,
      "msgRcvd":29285,
      "msgSent":29285,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"1d00h24m",
      "peerUptimeMsec":87873000,
      "prefixReceivedCount":1,
      "state":"Established",
      "idType":"ipv6"
    },
    "fd00::5":{
      "remoteAs":64514,
      "version":4,
      "msgRcvd":0,
      "msgSent":0,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"never",
      "peerUptimeMsec":0,
      "prefixReceivedCount":0,
      "state":"Active",
      "idType":"ipv6"
      }
  },
  "totalPeers":2,
  "dynamicPeers":0,
  "bestPath":{
    "multiPathRelax":"false"
  }
}
,
"red":{
  "routerId":"192.168.1.1",
  "as":64612,
  "vrfId":0,
  "vrfName":"default",
  "tableVersion":6,
  "ribCount":3,
  "ribMemory":456,
  "peerCount":2,
  "peerMemory":59904,
  "peers":{
    "fd00::101":{
      "remoteAs":64613,
      "version":4,
      "msgRcvd":29285,
      "msgSent":29285,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"1d00h24m",
      "peerUptimeMsec":87873000,
      "prefixReceivedCount":1,
      "state":"Established",
      "idType":"ipv6"
    },
    "fd00::105":{
      "remoteAs":64614,
      "version":4,
      "msgRcvd":0,
      "msgSent":0,
      "tableVersion":0,
      "outq":0,
      "inq":0,
      "peerUptime":"never",
      "peerUptimeMsec":0,
      "prefixReceivedCount":0,
      "state":"Active",
      "idType":"ipv6"
      }
  },
  "totalPeers":2,
  "dynamicPeers":0,
  "bestPath":{
    "multiPathRelax":"false"
  }
}
}
`)
	expectedBGPMetrics = map[string]float64{
		"frr_bgp_message_input_total{address_family=ipv4unicast,peer=192.168.0.2,vrf=default}":  100,
		"frr_bgp_message_input_total{address_family=ipv4unicast,peer=192.168.0.3,vrf=default}":  0,
		"frr_bgp_message_input_total{address_family=ipv4unicast,peer=192.168.1.2,vrf=red}":      100,
		"frr_bgp_message_input_total{address_family=ipv4unicast,peer=192.168.1.3,vrf=red}":      200,
		"frr_bgp_message_input_total{address_family=ipv6unicast,peer=fd00::1,vrf=default}":      29285,
		"frr_bgp_message_input_total{address_family=ipv6unicast,peer=fd00::101,vrf=red}":        29285,
		"frr_bgp_message_input_total{address_family=ipv6unicast,peer=fd00::105,vrf=red}":        0,
		"frr_bgp_message_input_total{address_family=ipv6unicast,peer=fd00::5,vrf=default}":      0,
		"frr_bgp_message_output_total{address_family=ipv4unicast,peer=192.168.0.2,vrf=default}": 100,
		"frr_bgp_message_output_total{address_family=ipv4unicast,peer=192.168.0.3,vrf=default}": 0,
		"frr_bgp_message_output_total{address_family=ipv4unicast,peer=192.168.1.2,vrf=red}":     100,
		"frr_bgp_message_output_total{address_family=ipv4unicast,peer=192.168.1.3,vrf=red}":     200,
		"frr_bgp_message_output_total{address_family=ipv6unicast,peer=fd00::1,vrf=default}":     29285,
		"frr_bgp_message_output_total{address_family=ipv6unicast,peer=fd00::101,vrf=red}":       29285,
		"frr_bgp_message_output_total{address_family=ipv6unicast,peer=fd00::105,vrf=red}":       0,
		"frr_bgp_message_output_total{address_family=ipv6unicast,peer=fd00::5,vrf=default}":     0,
		"frr_bgp_peer_groups_memory_bytes{address_family=ipv4unicast,vrf=default}":              0,
		"frr_bgp_peer_groups_memory_bytes{address_family=ipv4unicast,vrf=red}":                  0,
		"frr_bgp_peer_groups_memory_bytes{address_family=ipv6unicast,vrf=default}":              0,
		"frr_bgp_peer_groups_memory_bytes{address_family=ipv6unicast,vrf=red}":                  0,
		"frr_bgp_peer_groups{address_family=ipv4unicast,vrf=default}":                           0,
		"frr_bgp_peer_groups{address_family=ipv4unicast,vrf=red}":                               0,
		"frr_bgp_peer_groups{address_family=ipv6unicast,vrf=default}":                           0,
		"frr_bgp_peer_groups{address_family=ipv6unicast,vrf=red}":                               0,
		"frr_bgp_peer_up{address_family=ipv4unicast,peer=192.168.0.2,vrf=default}":              1,
		"frr_bgp_peer_up{address_family=ipv4unicast,peer=192.168.0.3,vrf=default}":              0,
		"frr_bgp_peer_up{address_family=ipv4unicast,peer=192.168.1.2,vrf=red}":                  1,
		"frr_bgp_peer_up{address_family=ipv4unicast,peer=192.168.1.3,vrf=red}":                  0,
		"frr_bgp_peer_up{address_family=ipv6unicast,peer=fd00::1,vrf=default}":                  1,
		"frr_bgp_peer_up{address_family=ipv6unicast,peer=fd00::101,vrf=red}":                    1,
		"frr_bgp_peer_up{address_family=ipv6unicast,peer=fd00::105,vrf=red}":                    0,
		"frr_bgp_peer_up{address_family=ipv6unicast,peer=fd00::5,vrf=default}":                  0,
		"frr_bgp_peer_uptime_seconds{address_family=ipv4unicast,peer=192.168.0.2,vrf=default}":  10,
		"frr_bgp_peer_uptime_seconds{address_family=ipv4unicast,peer=192.168.0.3,vrf=default}":  0,
		"frr_bgp_peer_uptime_seconds{address_family=ipv4unicast,peer=192.168.1.2,vrf=red}":      20,
		"frr_bgp_peer_uptime_seconds{address_family=ipv4unicast,peer=192.168.1.3,vrf=red}":      0,
		"frr_bgp_peer_uptime_seconds{address_family=ipv6unicast,peer=fd00::1,vrf=default}":      87873,
		"frr_bgp_peer_uptime_seconds{address_family=ipv6unicast,peer=fd00::101,vrf=red}":        87873,
		"frr_bgp_peer_uptime_seconds{address_family=ipv6unicast,peer=fd00::105,vrf=red}":        0,
		"frr_bgp_peer_uptime_seconds{address_family=ipv6unicast,peer=fd00::5,vrf=default}":      0,
		"frr_bgp_peers_memory_usage_bytes{address_family=ipv4unicast,vrf=default}":              39936,
		"frr_bgp_peers_memory_usage_bytes{address_family=ipv4unicast,vrf=red}":                  39936,
		"frr_bgp_peers_memory_usage_bytes{address_family=ipv6unicast,vrf=default}":              59904,
		"frr_bgp_peers_memory_usage_bytes{address_family=ipv6unicast,vrf=red}":                  59904,
		"frr_bgp_peers{address_family=ipv4unicast,vrf=default}":                                 2,
		"frr_bgp_peers{address_family=ipv4unicast,vrf=red}":                                     2,
		"frr_bgp_peers{address_family=ipv6unicast,vrf=default}":                                 2,
		"frr_bgp_peers{address_family=ipv6unicast,vrf=red}":                                     2,
		"frr_bgp_prefixes_active{address_family=ipv4unicast,peer=192.168.0.2,vrf=default}":      0,
		"frr_bgp_prefixes_active{address_family=ipv4unicast,peer=192.168.0.3,vrf=default}":      2,
		"frr_bgp_prefixes_active{address_family=ipv4unicast,peer=192.168.1.2,vrf=red}":          2,
		"frr_bgp_prefixes_active{address_family=ipv4unicast,peer=192.168.1.3,vrf=red}":          0,
		"frr_bgp_prefixes_active{address_family=ipv6unicast,peer=fd00::1,vrf=default}":          1,
		"frr_bgp_prefixes_active{address_family=ipv6unicast,peer=fd00::101,vrf=red}":            1,
		"frr_bgp_prefixes_active{address_family=ipv6unicast,peer=fd00::105,vrf=red}":            0,
		"frr_bgp_prefixes_active{address_family=ipv6unicast,peer=fd00::5,vrf=default}":          0,
		"frr_bgp_rib_entries{address_family=ipv4unicast,vrf=default}":                           1,
		"frr_bgp_rib_entries{address_family=ipv4unicast,vrf=red}":                               0,
		"frr_bgp_rib_entries{address_family=ipv6unicast,vrf=default}":                           3,
		"frr_bgp_rib_entries{address_family=ipv6unicast,vrf=red}":                               3,
		"frr_bgp_rib_memory_usage_bytes{address_family=ipv4unicast,vrf=default}":                64,
		"frr_bgp_rib_memory_usage_bytes{address_family=ipv4unicast,vrf=red}":                    0,
		"frr_bgp_rib_memory_usage_bytes{address_family=ipv6unicast,vrf=default}":                456,
		"frr_bgp_rib_memory_usage_bytes{address_family=ipv6unicast,vrf=red}":                    456,
	}
)

func TestProcessBGPSummary(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processBGPSummary(ch, bgpSumV4Unicast, "ipv4unicast"); err != nil {
		t.Errorf("error calling processBGPSummary ipv4unicast: %s", err)
	}
	if err := processBGPSummary(ch, bgpSumV6Unicast, "ipv6unicast"); err != nil {
		t.Errorf("error calling processBGPSummary ipv6unicast: %s", err)
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
		if expectedMetricVal, ok := expectedBGPMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedBGPMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}
