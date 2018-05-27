package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/tynany/frr_exporter/collector"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9342").String()
	telemetryPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()

	collectorEnabledState = map[collector.Collector]*bool{}

	collectors = []collector.Collector{
		new(collector.BGPCollector),
		new(collector.OSPFCollector),
	}
)

func handler(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()
	nc := collector.NewExporter(&collectorEnabledState)
	registry.Register(nc)

	gatheres := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	handlerOpts := promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError,
	}
	promhttp.HandlerFor(gatheres, handlerOpts).ServeHTTP(w, r)
}

func parseFlags() {
	for _, collector := range collectors {
		enabledByDefault := "false"
		if collector.EnabledByDefault() {
			enabledByDefault = "true"
		}
		collectorEnabledFlag := kingpin.Flag(fmt.Sprintf("collector.%s", collector.Name()), collector.Help()).Default(enabledByDefault).Bool()
		collectorEnabledState[collector] = collectorEnabledFlag
	}
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("frr_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
}

func main() {
	parseFlags()

	http.HandleFunc(*telemetryPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>FRR Exporter</title></head>
			<body>
			<h1>FRR Exporter</h1>
			<p><a href="` + *telemetryPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Starting frr_exporter on", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
