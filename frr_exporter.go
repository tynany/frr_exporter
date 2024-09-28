package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/tynany/frr_exporter/collector"
)

var (
	telemetryPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	webFlagConfig = kingpinflag.AddFlags(kingpin.CommandLine, ":9342")
)

func main() {
	promslogConfig := &promslog.Config{}

	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print("frr_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promslogConfig)

	prometheus.MustRegister(versioncollector.NewCollector("frr_exporter"))

	logger.Info("Starting frr_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	nc, err := collector.NewExporter(logger)
	if err != nil {
		panic(fmt.Sprintf("Couldn't create collector: %s", err))
	}

	prometheus.MustRegister(nc)

	http.Handle(*telemetryPath, promhttp.Handler())
	if *telemetryPath != "/" && *telemetryPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "FRR Exporter",
			Description: "Prometheus Exporter for FRRouting daemon",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{Address: *telemetryPath, Text: "Metrics"},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	server := &http.Server{}
	if err := web.ListenAndServe(server, webFlagConfig, logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
