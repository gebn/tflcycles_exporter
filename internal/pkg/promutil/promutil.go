package promutil

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HandlerOptsWithLogger returns an OpenMetrics-enabled set of handler options,
// which will log at error level on the provided logger.
func HandlerOptsWithLogger(logger *zap.Logger) promhttp.HandlerOpts {
	stdLog, err := zap.NewStdLogAt(logger, zapcore.ErrorLevel)
	if err != nil {
		// NewStdLogAt() will only throw if the log level above is invalid. The
		// tests will catch this if it happens.
		panic(err)
	}
	return promhttp.HandlerOpts{
		ErrorLog:          stdLog,
		EnableOpenMetrics: true,
	}
}
