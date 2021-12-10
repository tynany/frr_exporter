package collector

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/tynany/frr_exporter/internal/frrsockets"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	metric_namespace  = "frr"
	enabledByDefault  = true
	disabledByDefault = false
)

var (
	socketConn          *frrsockets.Connection
	frrTotalScrapeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metric_namespace,
		Name:      "scrapes_total",
		Help:      "Total number of times FRR has been scraped.",
	})
	frrLabels = []string{"collector"}
	frrDesc   = map[string]*prometheus.Desc{
		"frrScrapeDuration": promDesc("scrape_duration_seconds", "Time it took for a collector's scrape to complete.", frrLabels),
		"frrCollectorUp":    promDesc("collector_up", "Whether the collector's last scrape was successful (1 = successful, 0 = unsuccessful).", frrLabels),
		"frrUp":             promDesc("up", "Whether FRR is currently up.", nil),
	}

	socketDirPath = kingpin.Flag("frr.socket.dir-path", "Path of of the localstatedir containing each daemon's Unix socket.").Default("/var/run/frr").String()
	socketTimeout = kingpin.Flag("frr.socket.timeout", "Timeout when connecting to the FRR daemon Unix sockets").Default("20s").Duration()

	factories              = make(map[string]func(logger log.Logger) (Collector, error))
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]*bool)
)

func registerCollector(name string, enabledByDefaultStatus bool, factory func(logger log.Logger) (Collector, error)) {
	defaultState := "disabled"
	if enabledByDefaultStatus {
		defaultState = "enabled"
	}

	factories[name] = factory
	collectorState[name] = kingpin.Flag(fmt.Sprintf("collector.%s", name), fmt.Sprintf("Enable the %s collector (default: %s).", name, defaultState)).Default(strconv.FormatBool(enabledByDefaultStatus)).Bool()
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Update metrics and sends to the Prometheus.Metric channel.
	Update(ch chan<- prometheus.Metric) error
}

// Exporter collects all collector metrics, implemented as per the prometheus.Collector interface.
type Exporter struct {
	Collectors map[string]Collector
	logger     log.Logger
}

// NewExporter returns a new Exporter.
func NewExporter(logger log.Logger) (*Exporter, error) {
	collectors := make(map[string]Collector)

	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()

	socketConn = frrsockets.NewConnection(*socketDirPath, *socketTimeout)

	for name, enabled := range collectorState {
		if !*enabled {
			continue
		}
		if collector, exists := initiatedCollectors[name]; exists {
			collectors[name] = collector
		} else {
			collector, err := factories[name](log.With(logger, "collector", name))
			if err != nil {
				return nil, err
			}
			collectors[name] = collector
			initiatedCollectors[name] = collector
		}
	}
	return &Exporter{
		Collectors: collectors,
		logger:     logger,
	}, nil
}

// Collect implemented as per the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	frrTotalScrapeCount.Inc()
	ch <- frrTotalScrapeCount

	wg := &sync.WaitGroup{}
	wg.Add(len(e.Collectors))
	for name, collector := range e.Collectors {
		go runCollector(ch, name, collector, wg, e.logger)
	}
	wg.Wait()
}

func runCollector(ch chan<- prometheus.Metric, name string, collector Collector, wg *sync.WaitGroup, logger log.Logger) {
	defer wg.Done()

	startTime := time.Now()
	err := collector.Update(ch)
	scrapeDurationSeconds := time.Since(startTime).Seconds()

	ch <- prometheus.MustNewConstMetric(frrDesc["frrScrapeDuration"], prometheus.GaugeValue, float64(scrapeDurationSeconds), name)

	success := 0.0
	if err != nil {
		level.Error(logger).Log("msg", "collector scrape failed", "name", name, "duration_seconds", scrapeDurationSeconds, "err", err)
	} else {
		level.Debug(logger).Log("msg", "collector succeeded", "name", name, "duration_seconds", scrapeDurationSeconds)
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(frrDesc["frrCollectorUp"], prometheus.GaugeValue, success, name)
}

// Describe implemented as per the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range frrDesc {
		ch <- desc
	}
}

func promDesc(metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(metric_namespace+"_"+metricName, metricDescription, labels, nil)
}

func colPromDesc(subsystem string, metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(metric_namespace, subsystem, metricName), metricDescription, labels, nil)
}

func newGauge(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(descName, prometheus.GaugeValue, metric, labels...)
}

func newCounter(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(descName, prometheus.CounterValue, metric, labels...)
}

func cmdOutputProcessError(cmd, output string, err error) error {
	return fmt.Errorf("cannot process output of %s: %w: command output: %s", cmd, err, output)
}
