package collector

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const statusSubsystem = "status"

func init() {
	registerCollector(statusSubsystem, enabledByDefault, NewStatusCollector)
}

type statusCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

// NewStatusCollector collects FRR status metrics, implemented as per the Collector interface.
func NewStatusCollector(logger *slog.Logger) (Collector, error) {
	return &statusCollector{logger: logger, descriptions: getStatusDesc()}, nil
}

func getStatusDesc() map[string]*prometheus.Desc {
	labels := []string{"version", "os"}

	return map[string]*prometheus.Desc{
		"up": colPromDesc(statusSubsystem, "up", "FRR status (1 = up and responding, 0 = down or unreachable)", labels),
	}
}

// Update implemented as per the Collector interface
func (c *statusCollector) Update(ch chan<- prometheus.Metric) error {
	cmd := "show version"
	output, err := executeZebraCommand(cmd)

	var version, os string
	var status float64

	if err != nil {
		c.logger.Error("failed to execute show version command", "err", err)
		version = "unknown"
		os = "unknown"
		status = 0
	} else {
		var parseErr error
		version, os, parseErr = processStatusVersion(output)
		if parseErr != nil {
			c.logger.Error("failed to parse show version output", "err", parseErr, "output", string(output))
			version = "unknown"
			os = "unknown"
			status = 0
		} else {
			status = 1
		}
	}

	newGauge(ch, c.descriptions["up"], status, version, os)

	// Always return nil - we always emit a metric to indicate status
	return nil
}

func processStatusVersion(output []byte) (string, string, error) {
	text := string(output)
	lines := strings.Split(text, "\n")

	if len(lines) == 0 {
		return "", "", fmt.Errorf("empty output")
	}

	firstLine := lines[0]

	// Extract version using regex: FRRouting VERSION (...)
	versionRegex := regexp.MustCompile(`FRRouting (\S+)`)
	versionMatch := versionRegex.FindStringSubmatch(firstLine)
	if len(versionMatch) < 2 {
		return "", "", fmt.Errorf("could not extract version from: %s", firstLine)
	}
	version := versionMatch[1]

	// Extract OS using regex: on OS.
	osRegex := regexp.MustCompile(`on (.+)\.$`)
	osMatch := osRegex.FindStringSubmatch(firstLine)
	if len(osMatch) < 2 {
		return "", "", fmt.Errorf("could not extract OS from: %s", firstLine)
	}
	os := osMatch[1]

	return version, os, nil
}
