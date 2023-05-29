// Package exporter implements a Prometheus exporter for TfL Cycles
// availability.
package exporter

import (
	"net/http"
	"time"

	"github.com/gebn/tflcycles_exporter/internal/pkg/promutil"
	"github.com/gebn/tflcycles_exporter/internal/pkg/tflcycles"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	fetchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "tflcycles_exporter_fetch_duration_seconds",
		Help: "Observes the end-to-end duration of fetch operations.",
	})
	fetchFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tflcycles_exporter_fetch_failures_total",
		Help: "Counts the number of fetch operations that have failed.",
	})
)

// Exporter is an http.Handler that will respond to Prometheus scrape requests
// with information about stations' dock and cycle availability. Create
// instances with NewExporter().
type Exporter struct {
	Logger *zap.Logger
	Client *tflcycles.Client

	handlerOpts promhttp.HandlerOpts
}

func NewExporter(logger *zap.Logger, client *tflcycles.Client) *Exporter {
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
		e.Logger.Error("failed to fetch station availabilities", zap.Error(err))
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
