package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func readTestFixture(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("cannot read test fixture: %v", err)
	}
	return data
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

func collectMetrics(t *testing.T, ch <-chan prometheus.Metric) map[string]float64 {
	got := make(map[string]float64)
	re := regexp.MustCompile(`.*fqName: "(.*)", help:.*`)

	for m := range ch {
		var dtoM dto.Metric
		if err := m.Write(&dtoM); err != nil {
			t.Errorf("Write(): %v", err)
			continue
		}

		// build label strings WITHOUT quotes
		var lbls []string
		for _, l := range dtoM.GetLabel() {
			lbls = append(lbls, fmt.Sprintf("%s=%s", l.GetName(), l.GetValue()))
		}
		// sort them so the order is deterministic: area,iface,instance,vrf
		sort.Strings(lbls)

		// grab the numeric value
		var v float64
		if c := dtoM.GetCounter(); c != nil {
			v = c.GetValue()
		} else if g := dtoM.GetGauge(); g != nil {
			v = g.GetValue()
		}

		// extract the metric name from the Desc() text
		name := re.FindStringSubmatch(m.Desc().String())[1]
		key := fmt.Sprintf("%s{%s}", name, strings.Join(lbls, ","))
		got[key] = v
	}
	return got
}
