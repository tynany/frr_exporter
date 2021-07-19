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
	vrrpJson = []byte(`[
    {
      "vrid":1,
      "version":3,
      "autoconfigured":false,
      "shutdown":false,
      "preemptMode":true,
      "acceptMode":true,
      "interface":"gw_extnet",
      "advertisementInterval":1000,
      "v4":{
        "interface":"extnet_v4_1",
        "vmac":"00:00:5e:00:01:01",
        "primaryAddress":"",
        "status":"Backup",
        "effectivePriority":100,
        "masterAdverInterval":1000,
        "skewTime":600,
        "masterDownInterval":3600,
        "stats":{
          "adverTx":6,
          "adverRx":1548196,
          "garpTx":4,
          "transitions":9
        },
        "addresses":[
          "192.0.2.1"
        ]
      },
      "v6":{
        "interface":"extnet_v6_1",
        "vmac":"00:00:5e:00:02:01",
        "primaryAddress":"::",
        "status":"Backup",
        "effectivePriority":100,
        "masterAdverInterval":1000,
        "skewTime":600,
        "masterDownInterval":3600,
        "stats":{
          "adverTx":2,
          "adverRx":1548195,
          "neighborAdverTx":5,
          "transitions":11
        },
        "addresses":[
          "2001:DB8:2c02::1"
        ]
      }
    },
    {
      "vrid":2,
      "version":3,
      "autoconfigured":false,
      "shutdown":false,
      "preemptMode":true,
      "acceptMode":true,
      "interface":"gw_extnet",
      "advertisementInterval":1000,
      "v4":{
        "interface":"extnet_v4_2",
        "vmac":"00:00:5e:00:01:02",
        "primaryAddress":"192.0.2.3",
        "status":"Master",
        "effectivePriority":200,
        "masterAdverInterval":1000,
        "skewTime":210,
        "masterDownInterval":3210,
        "stats":{
          "adverTx":1548210,
          "adverRx":4,
          "garpTx":1,
          "transitions":2
        },
        "addresses":[
          "192.0.2.1"
        ]
      },
      "v6":{
        "interface":"",
        "vmac":"00:00:5e:00:02:02",
        "primaryAddress":"::",
        "status":"Initialize",
        "effectivePriority":200,
        "masterAdverInterval":0,
        "skewTime":0,
        "masterDownInterval":0,
        "stats":{
          "adverTx":0,
          "adverRx":0,
          "neighborAdverTx":0,
          "transitions":0
        },
        "addresses":[
        ]
      }
    }
  ]
`)

	expectedVRRPMetrics = map[string]float64{
		"frr_vrrp_adverRx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":           1548196,
		"frr_vrrp_adverRx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":           4.0,
		"frr_vrrp_adverRx_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":                      0.0,
		"frr_vrrp_adverRx_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":           1548195,
		"frr_vrrp_adverTx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":           6,
		"frr_vrrp_adverTx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":           1548210,
		"frr_vrrp_adverTx_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":                      0,
		"frr_vrrp_adverTx_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":           2,
		"frr_vrrp_garpTx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":            4,
		"frr_vrrp_garpTx_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":            1,
		"frr_vrrp_neighborAdverTx_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":              0,
		"frr_vrrp_neighborAdverTx_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":   5,
		"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}": 9,
		"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}": 2,
		"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":            0,
		"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}": 11,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Backup,subinterface=extnet_v4_1,vrid=1}":       1,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Backup,subinterface=extnet_v4_2,vrid=2}":       0,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Initialize,subinterface=extnet_v4_1,vrid=1}": 0,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Initialize,subinterface=extnet_v4_2,vrid=2}": 0,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Master,subinterface=extnet_v4_1,vrid=1}":       0,
		"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Master,subinterface=extnet_v4_2,vrid=2}":       1,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Backup,subinterface=,vrid=2}":                  0,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Backup,subinterface=extnet_v6_1,vrid=1}":       1,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Initialize,subinterface=,vrid=2}":            1,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Initialize,subinterface=extnet_v6_1,vrid=1}": 0,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Master,subinterface=,vrid=2}":                  0,
		"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Master,subinterface=extnet_v6_1,vrid=1}":       0,
	}
)

func TestProcessVRRPInfo(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processVRRPInfo(ch, vrrpJson); err != nil {
		t.Errorf("error calling processVRRPInfo: %s", err)
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
		if expectedMetricVal, ok := expectedVRRPMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedVRRPMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}