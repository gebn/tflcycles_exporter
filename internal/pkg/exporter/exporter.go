// Package exporter implements a Prometheus exporter for TfL Cycles
// availability.
package exporter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gebn/tflcycles_exporter/internal/pkg/bikepoint"
	"github.com/gebn/tflcycles_exporter/internal/pkg/promutil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	fetchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "tflcycles_exporter_fetch_duration_seconds",
		Help: "The end-to-end duration of BikePoint interactions, including any retries.",
		// These are copied from the tflcycles client histogram, because in
		// practice, that's the latency of the end-to-end scrape.
		Buckets: prometheus.ExponentialBuckets(.5, 1.223, 10), // 3.06
	})
	fetchFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tflcycles_exporter_fetch_failures_total",
		Help: "The number of BikePoint interactions that failed, even after any retrying.",
	})
)

// Exporter is an http.Handler that will respond to Prometheus scrape requests
// with information about stations' dock and cycle availability. Create
// instances with NewExporter().
type Exporter struct {
	Logger *slog.Logger
	Client *bikepoint.Client

	handlerOpts promhttp.HandlerOpts
}

func NewExporter(logger *slog.Logger, client *bikepoint.Client) *Exporter {
	return &Exporter{
		Logger:      logger,
		Client:      client,
		handlerOpts: promutil.HandlerOptsWithLogger(logger),
	}
}

func (e Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	start := time.Now()
	stationAvailabilities, err := e.Client.FetchStationAvailabilities(ctx)
	elapsed := time.Since(start)
	fetchDuration.Observe(elapsed.Seconds())
	if err != nil {
		fetchFailures.Inc()
		e.Logger.ErrorContext(ctx, "failed to fetch station availabilities",
			slog.String("error", err.Error()))
		// Force to nil, even if we received a non-nil slice.
		stationAvailabilities = nil
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(ScrapeCollector{
		Success:  stationAvailabilities != nil,
		Duration: elapsed,
	})
	if stationAvailabilities != nil {
		reg.MustRegister(StationAvailabilitiesCollector{
			StationAvailabilities: stationAvailabilities,
		})
	}
	promhttp.HandlerFor(reg, e.handlerOpts).ServeHTTP(w, r)
}
