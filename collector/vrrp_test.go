package collector

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var expectedVRRPMetrics = map[string]float64{
	"frr_vrrp_advertisements_received_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":      1548196,
	"frr_vrrp_advertisements_received_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":      4.0,
	"frr_vrrp_advertisements_received_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":                 0.0,
	"frr_vrrp_advertisements_received_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":      1548195,
	"frr_vrrp_advertisements_sent_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":          6,
	"frr_vrrp_advertisements_sent_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":          1548210,
	"frr_vrrp_advertisements_sent_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":                     0,
	"frr_vrrp_advertisements_sent_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":          2,
	"frr_vrrp_gratuitous_arp_sent_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":          4,
	"frr_vrrp_gratuitous_arp_sent_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":          1,
	"frr_vrrp_neighbor_advertisements_sent_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":            0,
	"frr_vrrp_neighbor_advertisements_sent_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}": 5,
	"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_1,vrid=1}":            9,
	"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v4,subinterface=extnet_v4_2,vrid=2}":            2,
	"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v6,subinterface=,vrid=2}":                       0,
	"frr_vrrp_state_transitions_total{interface=gw_extnet,proto=v6,subinterface=extnet_v6_1,vrid=1}":            11,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Backup,subinterface=extnet_v4_1,vrid=1}":                 1,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Backup,subinterface=extnet_v4_2,vrid=2}":                 0,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Initialize,subinterface=extnet_v4_1,vrid=1}":             0,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Initialize,subinterface=extnet_v4_2,vrid=2}":             0,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Master,subinterface=extnet_v4_1,vrid=1}":                 0,
	"frr_vrrp_state{interface=gw_extnet,proto=v4,state=Master,subinterface=extnet_v4_2,vrid=2}":                 1,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Backup,subinterface=,vrid=2}":                            0,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Backup,subinterface=extnet_v6_1,vrid=1}":                 1,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Initialize,subinterface=,vrid=2}":                        1,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Initialize,subinterface=extnet_v6_1,vrid=1}":             0,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Master,subinterface=,vrid=2}":                            0,
	"frr_vrrp_state{interface=gw_extnet,proto=v6,state=Master,subinterface=extnet_v6_1,vrid=1}":                 0,
}

func TestProcessVRRPInfo(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	if err := processVRRPInfo(ch, readTestFixture(t, "show_vrrp.json"), getVRRPDesc()); err != nil {
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
