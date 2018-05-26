package collector

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// The namespace used by all metrics.
const namespace = "frr"

var (
	frrTotalScrapeCount = 0.0
	frrTotalErrorCount  = 0

	frrScrapesTotal   = prometheus.NewDesc(namespace+"_scrapes_total", "Total number of times FRR has been scraped.", nil, nil)
	frrScrapeErrTotal = prometheus.NewDesc(namespace+"_scrape_errors_total", "Total number of errors from collector scrapes.", nil, nil)
	frrScrapeDuration = prometheus.NewDesc(namespace+"_scrape_duration_seconds", "Time it took for a collector's scrape to complete.", []string{"collector"}, nil)
	frrCollectorUp    = prometheus.NewDesc(namespace+"_collector_up", "Whether the collector's last scrape was successful (1 = successful, 0 = unsuccessful).", []string{"collector"}, nil)

	frrUp = prometheus.NewDesc(namespace+"_up", "Whether FRR is currently up.", nil, nil)

	frrVTYSHPath = kingpin.Flag("frr.vtysh.path", "Path of vtysh.").Default("/usr/bin/vtysh").String()
)

// Collector is the interface used by each FRR collector.
type Collector interface {
	// Returns a new collector.
	newCollector() Collector

	// Describe metrics.
	desc(ch chan<- *prometheus.Desc)

	// Scrape metrics.
	scrape(ch chan<- prometheus.Metric) error

	// What the collector does. Used to populate flag help.
	Help() string

	// Name of the collector. Used to populate the flag name.
	Name() string

	// Whether or not the collector is enabled by default. Used to populate flag default.
	EnabledByDefault() bool
}

// Exporter contains a slice of Collector interfaces.
type Exporter struct {
	collectors []Collector
}

// NewExporter creates a new exporter.
func NewExporter(collectorEnabledState *map[Collector]*bool) *Exporter {
	collectors := collectorsToScrape(collectorEnabledState)
	return &Exporter{collectors}
}

// Determine which collectors should be scraped.
func collectorsToScrape(collectorEnabledState *map[Collector]*bool) []Collector {
	collectors := []Collector{}
	for collector, enabled := range *collectorEnabledState {
		if *enabled {
			collectors = append(collectors, collector.newCollector())
		}
	}
	return collectors
}

// Describe implemented as per the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- frrScrapesTotal
	ch <- frrScrapeErrTotal
	ch <- frrUp
	ch <- frrScrapeDuration
	ch <- frrCollectorUp

	for _, collector := range e.collectors {
		collector.desc(ch)
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	frrTotalScrapeCount++
	ch <- prometheus.MustNewConstMetric(frrScrapesTotal, prometheus.CounterValue, frrTotalScrapeCount)

	errCh := make(chan int, 1024)
	wg := &sync.WaitGroup{}

	for _, collector := range e.collectors {
		wg.Add(1)
		go runCollector(ch, collector, wg, errCh)
	}

	wg.Wait()
	close(errCh)

	errCount := processErrors(errCh)

	frrTotalErrorCount = frrTotalErrorCount + errCount
	ch <- prometheus.MustNewConstMetric(frrScrapeErrTotal, prometheus.CounterValue, float64(frrTotalErrorCount))

	// If at least one collector is successfull we can assume FRR is running, otherwise assume FRR is not running. This is
	// cheaper than executing an FRR command and is a good enough method to determine whether FRR is up.
	frrState := 0.0
	if errCount < len(e.collectors) {
		frrState = 1
	}
	ch <- prometheus.MustNewConstMetric(frrUp, prometheus.GaugeValue, frrState)
}

func processErrors(errCh chan int) int {
	errors := 0
	for {
		_, more := <-errCh
		if !more {
			return errors
		}
		errors++
	}
}

func runCollector(ch chan<- prometheus.Metric, collector Collector, wg *sync.WaitGroup, errCh chan<- int) {
	defer wg.Done()
	collectorState := 1.0
	startTime := time.Now()

	if err := collector.scrape(ch); err != nil {
		collectorState = 0
		errCh <- 1
		log.Errorf("collector \"%s\" scrape failed: %s", collector.Name(), err)
	}
	ch <- prometheus.MustNewConstMetric(frrCollectorUp, prometheus.GaugeValue, collectorState, collector.Name())
	ch <- prometheus.MustNewConstMetric(frrScrapeDuration, prometheus.GaugeValue, float64(time.Since(startTime).Seconds()), collector.Name())
}
