package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gebn/tflcycles_exporter/internal/pkg/exporter"
	"github.com/gebn/tflcycles_exporter/internal/pkg/tflcycles"

	"github.com/gebn/go-stamp/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	buildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tflcycles_exporter_build_info",
			Help: "The version and commit of the running exporter. Always 1.",
		},
		[]string{"version", "commit"},
	)
)

func main() {
	if err := app(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func app(ctx context.Context) error {
	buildInfo.WithLabelValues(stamp.Version, stamp.Commit).Set(1)

	exporter := exporter.Exporter{
		Client: tflcycles.NewClient(http.DefaultClient,
			tflcycles.WithAppKey(os.Getenv("APP_KEY"))),
	}
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/stations", exporter)
	return http.ListenAndServe(":9722", nil)

	// help
	// version
	// listen address+port
}
