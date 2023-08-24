package exporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	up = prometheus.NewDesc(
		"tflcycles_up",
		"Whether the request to TfL's /BikePoint API succeeded.",
		nil, nil,
	)
	scrapeDurationSeconds = prometheus.NewDesc(
		"tflcycles_scrape_duration_seconds",
		"The amount of time it took to retrieve and parse the data for the scrape.",
		nil, nil,
	)
)

// ScrapeCollector is a prometheus.Collector yielding metrics about the
// fetching of data from the TfL BikePoint API.
type ScrapeCollector struct {
	Success bool
	time.Duration
}

func (ScrapeCollector) Describe(d chan<- *prometheus.Desc) {
	d <- up
	d <- scrapeDurationSeconds
}

func (c ScrapeCollector) Collect(m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		up,
		// CollectAndCompare() seems to recognise the _up suffix and insists it
		// is untyped.
		prometheus.UntypedValue,
		boolToFloat64(c.Success),
	)
	m <- prometheus.MustNewConstMetric(
		scrapeDurationSeconds,
		prometheus.GaugeValue,
		c.Duration.Seconds(),
	)
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
