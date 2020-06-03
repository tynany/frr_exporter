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

	evpnVniJson = []byte(`
    {
  "174374":{
    "vni":174374,
    "type":"L2",
    "vxlanIf":"ONTEP1_174374",
    "numMacs":42,
    "numArpNd":0,
    "numRemoteVteps":1,
    "tenantVrf":"default",
    "remoteVteps":[
      "10.0.0.13"
    ]
  },
  "172192":{
    "vni":172192,
    "type":"L2",
    "vxlanIf":"ONTEP1_172192",
    "numMacs":0,
    "numArpNd":23,
    "numRemoteVteps":1,
    "tenantVrf":"default",
    "remoteVteps":[
      "10.0.0.13"
    ]
  }
  }`)
	expectedBGPMetrics = map[string]float64{
		"frr_bgp_peer_groups_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                           0.0,
		"frr_bgp_peer_groups_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                               0.0,
		"frr_bgp_peer_groups_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                           0.0,
		"frr_bgp_peer_groups_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                               0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                          0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                              0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                          0.0,
		"frr_bgp_peer_groups_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                              0.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":        100.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":        0.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":            100.0,
		"frr_bgp_peer_message_received_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":            200.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":            29285.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":            0.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":              29285.0,
		"frr_bgp_peer_message_received_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":              0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":            100.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":            0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                100.0,
		"frr_bgp_peer_message_sent_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                200.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":                29285.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":                0.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":                  29285.0,
		"frr_bgp_peer_message_sent_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":                  0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}": 0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}": 2.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":     2.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":     0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":     1.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":     0.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":       1.0,
		"frr_bgp_peer_prefixes_received_count_total{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":       0.0,
		"frr_bgp_peers_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                 2.0,
		"frr_bgp_peers_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                     2.0,
		"frr_bgp_peers_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                                 2.0,
		"frr_bgp_peers_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                     2.0,
		"frr_bgp_peers_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                39936.0,
		"frr_bgp_peers_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                    39936.0,
		"frr_bgp_peers_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                                59904.0,
		"frr_bgp_peers_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                    59904.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":                         1.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":                         0.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                             1.0,
		"frr_bgp_peer_state{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                             0.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":                             1.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":                             0.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":                               1.0,
		"frr_bgp_peer_state{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":                               0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64512,peer=192.168.0.2,peer_as=64513,safi=unicast,vrf=default}":                10.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64512,peer=192.168.0.3,peer_as=64514,safi=unicast,vrf=default}":                0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64612,peer=192.168.1.2,peer_as=64613,safi=unicast,vrf=red}":                    20.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv4,local_as=64612,peer=192.168.1.3,peer_as=64614,safi=unicast,vrf=red}":                    0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64512,peer=fd00::1,peer_as=64513,safi=unicast,vrf=default}":                    87873.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64512,peer=fd00::5,peer_as=64514,safi=unicast,vrf=default}":                    0.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64612,peer=fd00::101,peer_as=64613,safi=unicast,vrf=red}":                      87873.0,
		"frr_bgp_peer_uptime_seconds{afi=ipv6,local_as=64612,peer=fd00::105,peer_as=64614,safi=unicast,vrf=red}":                      0.0,
		"frr_bgp_rib_count_total{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                   1.0,
		"frr_bgp_rib_count_total{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                       0.0,
		"frr_bgp_rib_count_total{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                                   3.0,
		"frr_bgp_rib_count_total{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                       3.0,
		"frr_bgp_rib_memory_bytes{afi=ipv4,local_as=64512,safi=unicast,vrf=default}":                                                  64.0,
		"frr_bgp_rib_memory_bytes{afi=ipv4,local_as=64612,safi=unicast,vrf=red}":                                                      0.0,
		"frr_bgp_rib_memory_bytes{afi=ipv6,local_as=64512,safi=unicast,vrf=default}":                                                  456.0,
		"frr_bgp_rib_memory_bytes{afi=ipv6,local_as=64612,safi=unicast,vrf=red}":                                                      456.0,
	}
	expectedBgpL2vpnMetrics = map[string]float64{
		"frr_bgp_l2vpn_evpn_num_arp_nd_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}":       23.000000,
		"frr_bgp_l2vpn_evpn_num_arp_nd_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}":       0.000000,
		"frr_bgp_l2vpn_evpn_num_macs_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}":         0.000000,
		"frr_bgp_l2vpn_evpn_num_macs_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}":         42.000000,
		"frr_bgp_l2vpn_evpn_num_remote_vteps_count_total{tenantVrf=default,type=L2,vni=172192,vxlanIf=ONTEP1_172192}": 1.000000,
		"frr_bgp_l2vpn_evpn_num_remote_vteps_count_total{tenantVrf=default,type=L2,vni=174374,vxlanIf=ONTEP1_174374}": 1.000000,
	}
)

func prepareMetrics(ch chan prometheus.Metric, t *testing.T) map[string]float64 {
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
	return gotMetrics
}

func compareMetrics(t *testing.T, gotMetrics map[string]float64, expectedMetrics map[string]float64) {
	for metricName, metricVal := range gotMetrics {
		if expectedMetricVal, ok := expectedMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}

func TestProcessBGPSummary(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processBGPSummary(ch, bgpSumV4Unicast, "ipv4", "unicast"); err != nil {
		t.Errorf("error calling processBGPSummary ipv4unicast: %s", err)
	}
	if err := processBGPSummary(ch, bgpSumV6Unicast, "ipv6", "unicast"); err != nil {
		t.Errorf("error calling processBGPSummary ipv6unicast: %s", err)
	}
	close(ch)

	gotMetrics := prepareMetrics(ch, t)
	compareMetrics(t, gotMetrics, expectedBGPMetrics)
}

func TestProcessBgpL2vpnEvpnSummary(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processBgpL2vpnEvpnSummary(ch, evpnVniJson); err != nil {
		t.Errorf("error calling processBgpL2vpnEvpnSummary: %s", err)
	}
	close(ch)

	gotMetrics := prepareMetrics(ch, t)
	compareMetrics(t, gotMetrics, expectedBgpL2vpnMetrics)
}
