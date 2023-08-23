package promutil

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HandlerOptsWithLogger returns an OpenMetrics-enabled set of handler options,
// which will log at error level on the provided logger.
func HandlerOptsWithLogger(logger *slog.Logger) promhttp.HandlerOpts {
	return promhttp.HandlerOpts{
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		EnableOpenMetrics: true,
	}
}
