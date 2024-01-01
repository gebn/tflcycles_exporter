package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gebn/tflcycles_exporter/internal/pkg/bikepoint"
	"github.com/gebn/tflcycles_exporter/internal/pkg/exporter"
	"github.com/gebn/tflcycles_exporter/internal/pkg/promutil"

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

	showVersion := flag.Bool("version", false, "print the exporter version and exit")
	isDebug := flag.Bool("debug", false, "enable verbose, human-readable logging")
	listenAddr := flag.String("listen", ":9722", "the address and port to bind the web server to")
	flag.Parse()

	if *showVersion {
		fmt.Println(stamp.Summary())
		return nil
	}

	logger := buildLogger(*isDebug)
	slog.SetDefault(logger)

	indexHandler, err := buildIndexHandler(logger)
	if err != nil {
		return err
	}
	// This handler is also responsible for serving 404s.
	http.Handle("/", indexHandler)

	metricsHandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promutil.HandlerOptsWithLogger(logger),
	)
	http.Handle("/metrics", metricsHandler)

	stationsHandler := exporter.NewExporter(
		logger,
		bikepoint.NewClient(
			logger,
			http.DefaultClient,
			bikepoint.WithAppKey(os.Getenv("APP_KEY")),
		),
	)
	http.Handle("/stations", stationsHandler)

	return listenAndServe(ctx, logger, *listenAddr)
}

// buildLogger creates a suitable logger for the provided mode. If debugging is
// disabled, which will typically be the case, the logger is suitable for
// production: JSON format, info level. If debugging is enabled, we instead log
// for direct human interpretation, at debug level.
//
// Note the parameter to this function is not simply a log level, as it also
// influences the format.
func buildLogger(isDebug bool) *slog.Logger {
	if isDebug {
		// Uses the logfmt standard.
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, nil))
}

func listenAndServe(ctx context.Context, logger *slog.Logger, addr string) error {
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	logger.InfoContext(ctx, "listening", slog.String("addr", listener.Addr().String()))

	server := http.Server{
		ReadHeaderTimeout: time.Second,
		// This is above the max recommended scrape interval of 2m.
		IdleTimeout: 3 * time.Minute,
	}

	shutdown := make(chan error)
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		logger.InfoContext(ctx, "waiting for open connections to become idle")
		shutdown <- server.Shutdown(ctx)
	}()

	if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("web server terminated incorrectly: %w", err)
	}
	return <-shutdown
}
