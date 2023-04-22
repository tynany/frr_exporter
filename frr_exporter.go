package main

import (
	"fmt"
	inbuiltLog "log"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/tynany/frr_exporter/collector"
)

var (
	telemetryPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	webFlagConfig = kingpinflag.AddFlags(kingpin.CommandLine, ":9342")
)

func handler(logger log.Logger) http.Handler {
	registry := prometheus.NewRegistry()

	nc, err := collector.NewExporter(logger)
	if err != nil {
		panic(fmt.Sprintf("Couldn't create collector: %s", err))
	}

	if err := registry.Register(nc); err != nil {
		panic(fmt.Sprintf("Couldn't register collector: %s", err))
	}

	gatheres := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}

	handlerOpts := promhttp.HandlerOpts{
		ErrorLog:      inbuiltLog.New(log.NewStdlibAdapter(level.Error(logger)), "", 0),
		ErrorHandling: promhttp.ContinueOnError,
	}

	return promhttp.HandlerFor(gatheres, handlerOpts)
}

func main() {
	promlogConfig := &promlog.Config{}

	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("frr_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	prometheus.MustRegister(version.NewCollector("frr_exporter"))

	level.Info(logger).Log("msg", "Starting frr_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	http.Handle(*telemetryPath, handler(logger))
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
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	server := &http.Server{}
	if err := web.ListenAndServe(server, webFlagConfig, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
