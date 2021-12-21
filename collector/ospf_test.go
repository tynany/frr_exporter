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
	ospfInterfaceSum = []byte(`{
	  "default":{
	    "vrfName":"default",
	    "vrfId":0,
	    "swp1":{
	      "ifUp":true,
	      "ifIndex":4,
	      "mtuBytes":1500,
	      "bandwidthMbit":4294967295,
	      "ifFlags":"<UP,BROADCAST,RUNNING,MULTICAST>",
	      "ospfEnabled":true,
	      "ipAddress":"192.168.0.1",
	      "ipAddressPrefixlen":24,
	      "area":"0.0.0.0",
	      "routerId":"192.168.255.1",
	      "networkType":"BROADCAST",
	      "cost":1,
	      "transmitDelayMsecs":1000,
	      "state":"DR",
	      "priority":1,
	      "mcastMemberOspfAllRouters":true,
	      "mcastMemberOspfDesignatedRouters":true,
	      "timerMsecs":100,
	      "timerDeadMsecs":25,
	      "timerWaitMsecs":25,
	      "timerRetransmit":200,
	      "timerHelloInMsecs":7769,
	      "nbrCount":0,
	      "nbrAdjacentCount":0
	    },
	    "swp2":{
	      "ifUp":true,
	      "ifIndex":6,
	      "mtuBytes":1500,
	      "bandwidthMbit":4294967295,
	      "ifFlags":"<UP,BROADCAST,RUNNING,MULTICAST>",
	      "ospfEnabled":true,
	      "ipAddress":"192.168.2.1",
	      "ipAddressPrefixlen":24,
	      "area":"0.0.0.0",
	      "routerId":"192.168.255.1",
	      "networkType":"BROADCAST",
	      "cost":1,
	      "transmitDelayMsecs":1000,
	      "state":"DR",
	      "priority":1,
	      "bdrId":"1.1.1.1",
	      "bdrAddress":"192.168.1.2",
	      "networkLsaSequence":2147483717,
	      "mcastMemberOspfAllRouters":true,
	      "mcastMemberOspfDesignatedRouters":true,
	      "timerMsecs":100,
	      "timerDeadMsecs":25,
	      "timerWaitMsecs":25,
	      "timerRetransmit":200,
	      "timerHelloInMsecs":7769,
	      "nbrCount":1,
	      "nbrAdjacentCount":1
	    }
	  },
	  "red":{
	    "vrfName":"red",
	    "vrfId":0,
	    "swp3":{
	      "ifUp":true,
	      "ifIndex":4,
	      "mtuBytes":1500,
	      "bandwidthMbit":4294967295,
	      "ifFlags":"<UP,BROADCAST,RUNNING,MULTICAST>",
	      "ospfEnabled":true,
	      "ipAddress":"192.168.10.1",
	      "ipAddressPrefixlen":24,
	      "area":"0.0.0.0",
	      "routerId":"192.168.255.1",
	      "networkType":"BROADCAST",
	      "cost":1,
	      "transmitDelayMsecs":1000,
	      "state":"DR",
	      "priority":1,
	      "mcastMemberOspfAllRouters":true,
	      "mcastMemberOspfDesignatedRouters":true,
	      "timerMsecs":100,
	      "timerDeadMsecs":25,
	      "timerWaitMsecs":25,
	      "timerRetransmit":200,
	      "timerHelloInMsecs":7769,
	      "nbrCount":0,
	      "nbrAdjacentCount":0
	    },
	    "swp4":{
	      "ifUp":true,
	      "ifIndex":6,
	      "mtuBytes":1500,
	      "bandwidthMbit":4294967295,
	      "ifFlags":"<UP,BROADCAST,RUNNING,MULTICAST>",
	      "ospfEnabled":true,
	      "ipAddress":"192.168.12.1",
	      "ipAddressPrefixlen":24,
	      "area":"0.0.0.0",
	      "routerId":"192.168.255.1",
	      "networkType":"BROADCAST",
	      "cost":1,
	      "transmitDelayMsecs":1000,
	      "state":"DR",
	      "priority":1,
	      "bdrId":"1.1.1.1",
	      "bdrAddress":"192.168.1.2",
	      "networkLsaSequence":2147483717,
	      "mcastMemberOspfAllRouters":true,
	      "mcastMemberOspfDesignatedRouters":true,
	      "timerMsecs":100,
	      "timerDeadMsecs":25,
	      "timerWaitMsecs":25,
	      "timerRetransmit":200,
	      "timerHelloInMsecs":7769,
	      "nbrCount":1,
	      "nbrAdjacentCount":1
	    },
		"peerlink.4094":{
		  "ifUp":true,
		  "ifIndex":62,
		  "mtuBytes":9000,
		  "bandwidthMbit":2000,
		  "ifFlags":"<UP,BROADCAST,RUNNING,MULTICAST>",
		  "ospfEnabled":true,
		  "ipAddress":"169.254.1.1",
		  "ipAddressPrefixlen":30,
		  "ospfIfType":"Broadcast",
		  "localIfUsed":"169.254.1.3",
		  "area":"0.0.0.75 [Stub]",
		  "routerId":"10.200.1.222",
		  "networkType":"BROADCAST",
		  "cost":50,
		  "transmitDelaySecs":1,
		  "state":"DR",
		  "priority":1,
		  "timerMsecs":10000,
		  "timerDeadSecs":40,
		  "timerWaitSecs":40,
		  "timerRetransmitSecs":5,
		  "timerPassiveIface":true,
		  "nbrCount":0,
		  "nbrAdjacentCount":0
		}
	  }
	}
`)
	expectedOSPFMetrics = map[string]float64{
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
)

func TestProcessOSPFInterface(t *testing.T) {
	ch := make(chan prometheus.Metric, len(expectedOSPFMetrics))
	if err := processOSPFInterface(ch, ospfInterfaceSum, getOSPFDesc(), 0); err != nil {
		t.Errorf("error calling processOSPFInterface ipv4unicast: %s", err)
	}

	// test for OSPF multiple instances
	*frrOSPFInstances = "1,2"
	for i := 1; i <= 2; i++ {
		if err := processOSPFInterface(ch, ospfInterfaceSum, getOSPFDesc(), i); err != nil {
			t.Errorf("error calling processOSPFInterface ipv4unicast: %s", err)
		}
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
		if expectedMetricVal, ok := expectedOSPFMetrics[metricName]; ok {
			if expectedMetricVal != metricVal {
				t.Errorf("metric %s expected value %v got %v", metricName, expectedMetricVal, metricVal)
			}

		} else {
			t.Errorf("unexpected metric: %s : %v", metricName, metricVal)
		}
	}

	for expectedMetricName, expectedMetricVal := range expectedOSPFMetrics {
		if _, ok := gotMetrics[expectedMetricName]; !ok {
			t.Errorf("missing metric: %s value %v", expectedMetricName, expectedMetricVal)
		}
	}
}
