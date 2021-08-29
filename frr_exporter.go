package main

import (
	"fmt"
	inbuiltLog "log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"

	"github.com/go-kit/log"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/tynany/frr_exporter/collector"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress   = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9342").String()
	telemetryPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	frrVTYSHPath    = kingpin.Flag("frr.vtysh.path", "Path of vtysh.").Default("/usr/bin/vtysh").String()
	frrVTYSHOptions = kingpin.Flag("frr.vtysh.options", "Additional options passed to vtysh.").Default("").String()
	frrVTYSHTimeout = kingpin.Flag("frr.vtysh.timeout", "The timeout when running vtysh commands (default: 20s).").Default("20s").String()
	frrVTYSHSudo    = kingpin.Flag("frr.vtysh.sudo", "Enable sudo when executing vtysh commands.").Bool()
	configFile      = kingpin.Flag("web.config", "[EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.").Default("").String()

	collectors = []*collector.Collector{}
)

func initCollectors(logger log.Logger) {
	bgp := collector.NewBGPCollector()
	collectors = append(collectors, &collector.Collector{
		Name:          bgp.Name(),
		PromCollector: bgp,
		Errors:        bgp,
		CLIHelper:     bgp,
		Logger:        logger,
	})
	ospf := collector.NewOSPFCollector()
	collectors = append(collectors, &collector.Collector{
		Name:          ospf.Name(),
		PromCollector: ospf,
		Errors:        ospf,
		CLIHelper:     ospf,
		Logger:        logger,
	})
	bgp6 := collector.NewBGP6Collector()
	collectors = append(collectors, &collector.Collector{
		Name:          bgp6.Name(),
		PromCollector: bgp6,
		Errors:        bgp6,
		CLIHelper:     bgp6,
		Logger:        logger,
	})
	bgpl2vpn := collector.NewBGPL2VPNCollector()
	collectors = append(collectors, &collector.Collector{
		Name:          bgpl2vpn.Name(),
		PromCollector: bgpl2vpn,
		Errors:        bgpl2vpn,
		CLIHelper:     bgpl2vpn,
		Logger:        logger,
	})
	bfd := collector.NewBFDCollector()
	collectors = append(collectors, &collector.Collector{
		Name:          bfd.Name(),
		PromCollector: bfd,
		Errors:        bfd,
		CLIHelper:     bfd,
		Logger:        logger,
	})
	vrrp := collector.NewVRRPCollector()
	collectors = append(collectors, &collector.Collector{
		Name:          vrrp.Name(),
		PromCollector: vrrp,
		Errors:        vrrp,
		CLIHelper:     vrrp,
		Logger:        logger,
	})
}

func handler(logger log.Logger) http.Handler {
	registry := prometheus.NewRegistry()
	enabledCollectors := []*collector.Collector{}

	for _, collector := range collectors {
		if *collector.Enabled {
			enabledCollectors = append(enabledCollectors, collector)
		}
	}
	ne := collector.NewExporter(enabledCollectors)
	ne.SetVTYSHPath(*frrVTYSHPath)

	// error checking is done as part of parseCLI
	frrTimeout, _ := time.ParseDuration(*frrVTYSHTimeout)
	ne.SetVTYSHTimeout(frrTimeout)
	ne.SetVTYSHSudo(*frrVTYSHSudo)

	if *frrVTYSHOptions != "" {
		ne.SetVTYSHOptions(*frrVTYSHOptions)
	}

	registry.Register(ne)

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

func parseCLI(promlogConfig *promlog.Config) error {
	for _, collector := range collectors {
		defaultState := "disabled"
		enabledByDefault := collector.CLIHelper.EnabledByDefault()
		if enabledByDefault {
			defaultState = "enabled"
		}
		flagName := fmt.Sprintf("collector.%s", collector.CLIHelper.Name())
		helpString := fmt.Sprintf("%s (default: %s).", collector.CLIHelper.Help(), defaultState)
		collector.Enabled = kingpin.Flag(flagName, helpString).Default(strconv.FormatBool(enabledByDefault)).Bool()
	}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("frr_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	if _, err := time.ParseDuration(*frrVTYSHTimeout); err != nil {
		return fmt.Errorf("invalid frr.vtysh.timeout flag: %s", err)
	}
	return nil
}

func main() {
	prometheus.MustRegister(version.NewCollector("frr_exporter"))

	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	initCollectors(logger)
	if err := parseCLI(promlogConfig); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "Starting frr_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)

	http.Handle(*telemetryPath, handler(logger))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>FRR Exporter</title></head>
			<body>
			<h1>FRR Exporter</h1>
			<p><a href="` + *telemetryPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	server := &http.Server{Addr: *listenAddress}
	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
