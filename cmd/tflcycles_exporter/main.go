package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gebn/tflcycles_exporter/internal/pkg/exporter"
	"github.com/gebn/tflcycles_exporter/internal/pkg/tflcycles"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := app(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func app(ctx context.Context) error {
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
