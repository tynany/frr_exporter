package collector

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// The namespace used by all metrics.
const namespace = "frr"

var (
	frrTotalScrapeCount = 0.0
	frrTotalErrorCount  = 0
	frrLabels           = []string{"collector"}
	frrDesc             = map[string]*prometheus.Desc{
		"frrScrapesTotal":   promDesc("scrapes_total", "Total number of times FRR has been scraped.", nil),
		"frrScrapeErrTotal": promDesc("scrape_errors_total", "Total number of errors from a collector.", frrLabels),
		"frrScrapeDuration": promDesc("scrape_duration_seconds", "Time it took for a collector's scrape to complete.", frrLabels),
		"frrCollectorUp":    promDesc("collector_up", "Whether the collector's last scrape was successful (1 = successful, 0 = unsuccessful).", frrLabels),
		"frrUp":             promDesc("up", "Whether FRR is currently up.", nil),
	}
	vtyshPath    string
	vtyshTimeout time.Duration
	vtyshSudo    bool
)

// CLIHelper is used to populate flags.
type CLIHelper interface {
	// What the collector does.
	Help() string

	// Name of the collector.
	Name() string

	// Whether or not the collector is enabled by default.
	EnabledByDefault() bool
}

// CollectErrors is used to collect collector errors.
type CollectErrors interface {
	// Returns any errors that were encounted during Collect.
	CollectErrors() []error

	// Returns the total number of errors encounter during app run duration.
	CollectTotalErrors() float64
}

// Exporters contains a slice of Collectors.
type Exporters struct {
	Collectors []*Collector
}

// Collector contains everything needed to collect from a collector.
type Collector struct {
	Enabled       *bool
	Name          string
	PromCollector prometheus.Collector
	Errors        CollectErrors
	CLIHelper     CLIHelper
}

// NewExporter returns an Exporters type containing a slice of Collectors.
func NewExporter(collectors []*Collector) *Exporters {
	return &Exporters{Collectors: collectors}
}

// SetVTYSHPath sets the path of vtysh.
func (e *Exporters) SetVTYSHPath(path string) {
	vtyshPath = path
}

// SetVTYSHTimeout sets the path of vtysh.
func (e *Exporters) SetVTYSHTimeout(timeout time.Duration) {
	vtyshTimeout = timeout
}

// SetVTYSHSudo sets the first command to execute vtysh if sudo is enabled.
func (e *Exporters) SetVTYSHSudo(enable bool) {
        vtyshSudo = enable
}

// Describe implemented as per the prometheus.Collector interface.
func (e *Exporters) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range frrDesc {
		ch <- desc
	}
	for _, collector := range e.Collectors {
		collector.PromCollector.Describe(ch)
	}
}

// Collect implemented as per the prometheus.Collector interface.
func (e *Exporters) Collect(ch chan<- prometheus.Metric) {
	frrTotalScrapeCount++
	ch <- prometheus.MustNewConstMetric(frrDesc["frrScrapesTotal"], prometheus.CounterValue, frrTotalScrapeCount)

	errCh := make(chan int, 1024)
	wg := &sync.WaitGroup{}
	for _, collector := range e.Collectors {
		wg.Add(1)
		go runCollector(ch, errCh, collector, wg)
	}
	wg.Wait()

	close(errCh)
	errCount := processErrors(errCh)

	// If at least one collector is successfull we can assume FRR is running, otherwise assume FRR is not running. This is
	// cheaper than executing an FRR command and is a good enough method to determine whether FRR is up.
	frrState := 0.0
	if errCount < len(e.Collectors) {
		frrState = 1
	}
	ch <- prometheus.MustNewConstMetric(frrDesc["frrUp"], prometheus.GaugeValue, frrState)
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

func runCollector(ch chan<- prometheus.Metric, errCh chan<- int, collector *Collector, wg *sync.WaitGroup) {
	defer wg.Done()
	startTime := time.Now()

	collector.PromCollector.Collect(ch)

	ch <- prometheus.MustNewConstMetric(frrDesc["frrScrapeErrTotal"], prometheus.GaugeValue, collector.Errors.CollectTotalErrors(), collector.Name)

	errors := collector.Errors.CollectErrors()
	if len(errors) > 0 {
		errCh <- 1
		ch <- prometheus.MustNewConstMetric(frrDesc["frrCollectorUp"], prometheus.GaugeValue, 0, collector.Name)
		for _, err := range errors {
			log.Errorf("collector \"%s\" scrape failed: %s", collector.Name, err)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(frrDesc["frrCollectorUp"], prometheus.GaugeValue, 1, collector.Name)
	}
	ch <- prometheus.MustNewConstMetric(frrDesc["frrScrapeDuration"], prometheus.GaugeValue, float64(time.Since(startTime).Seconds()), collector.Name)
}

func promDesc(metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(namespace+"_"+metricName, metricDescription, labels, nil)
}

func colPromDesc(subsystem string, metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, metricName), metricDescription, labels, nil)
}

func newGauge(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(descName, prometheus.GaugeValue, metric, labels...)
}

func newCounter(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(descName, prometheus.CounterValue, metric, labels...)
}
