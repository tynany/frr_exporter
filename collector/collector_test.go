package collector

import (
	"os"
	"path/filepath"
	"testing"
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
