package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	expectedStatusMetrics = map[string]float64{
		"frr_status_up{os=Linux(5.14.0-284.11.1.el9_2.x86_64),version=10.3.1}": 1,
	}

	expectedStatusMetricsDown = map[string]float64{
		"frr_status_up{os=unknown,version=unknown}": 0,
	}
)

func TestProcessStatusVersion(t *testing.T) {
	fixture := readTestFixture(t, "show_version.txt")

	version, os, err := processStatusVersion(fixture)
	if err != nil {
		t.Errorf("error calling processStatusVersion: %s", err)
	}

	expectedVersion := "10.3.1"
	if version != expectedVersion {
		t.Errorf("expected version %s, got %s", expectedVersion, version)
	}

	expectedOS := "Linux(5.14.0-284.11.1.el9_2.x86_64)"
	if os != expectedOS {
		t.Errorf("expected os %s, got %s", expectedOS, os)
	}
}

func TestProcessStatusVersionWithMetrics(t *testing.T) {
	fixture := readTestFixture(t, "show_version.txt")

	version, os, err := processStatusVersion(fixture)
	if err != nil {
		t.Errorf("error calling processStatusVersion: %s", err)
	}

	ch := make(chan prometheus.Metric, 1024)
	statusDesc := getStatusDesc()
	newGauge(ch, statusDesc["up"], 1, version, os)
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedStatusMetrics)
}

func TestProcessStatusVersionEmpty(t *testing.T) {
	_, _, err := processStatusVersion([]byte(""))
	if err == nil {
		t.Error("expected error for empty output, got nil")
	}
}

func TestProcessStatusVersionInvalid(t *testing.T) {
	_, _, err := processStatusVersion([]byte("Invalid output without version info"))
	if err == nil {
		t.Error("expected error for invalid output, got nil")
	}
}

func TestStatusCollectorUpdateFailure(t *testing.T) {
	ch := make(chan prometheus.Metric, 1024)
	statusDesc := getStatusDesc()

	// Simulate failure case by emitting status=0 with unknown labels
	newGauge(ch, statusDesc["up"], 0, "unknown", "unknown")
	close(ch)

	gotMetrics := collectMetrics(t, ch)
	compareMetrics(t, gotMetrics, expectedStatusMetricsDown)
}
