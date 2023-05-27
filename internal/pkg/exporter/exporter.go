// Package exporter implements a Prometheus exporter for TfL Cycles
// availability.
package exporter

import (
	"log"
	"net/http"
	"time"

	"github.com/gebn/tflcycles_exporter/internal/pkg/tflcycles"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	handlerOpts = promhttp.HandlerOpts{
		ErrorLog:          log.Default(),
		EnableOpenMetrics: true,
	}
)

type Exporter struct {
	Client *tflcycles.Client
}

func (e Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	start := time.Now()
	stationAvailabilities, err := e.Client.FetchStationAvailabilities(ctx)
	elapsed := time.Since(start)
	fetchDuration.Observe(elapsed.Seconds())
	if err != nil {
		fetchFailures.Inc()
		log.Printf("failed to fetch station availabilities: %v", err)
		// stationAvailabilities will be nil.
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
	promhttp.HandlerFor(reg, handlerOpts).ServeHTTP(w, r)
}
