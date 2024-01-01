package bikepoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gebn/go-stamp/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// This will observe a shorter value if a given request is terminated early
	// due to timeout.
	httpRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "tflcycles_bikepoint_http_request_duration_seconds",
		Help: "Observes the duration of all requests to /BikePoint, including response parsing.",
		// The last bucket should be just above our timeout.
		Buckets: prometheus.ExponentialBuckets(.2, 1.355, 10), // 3.08
	})
	// This is arguably redundant given the existence of retries, which
	// provides an indication of failures. This metric also does not correspond
	// to a single line of code.
	httpRequestFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tflcycles_bikepoint_http_request_failures_total",
		Help: "The number of BikePoint API requests that timed out or returned an invalid response",
	})
	httpRequestRetries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tflcycles_bikepoint_http_request_retries_total",
		Help: "The number of times we timed-out or received a 5xx error from /BikePoint, and retried.",
	})
)

// Client is used to interact with the BikePoint API. Create instances with
// NewClient().
//
// We do not implement station filtering here as there would be no point - we'd
// have to decode the JSON and then delete bits from the output. This is left
// to the user of the client.
type Client struct {

	// Logger will be used to record failed fetch attempts.
	Logger *slog.Logger

	// HTTPClient is the client used to make requests to the API. This must be
	// provided when calling NewClient().
	HTTPClient *http.Client

	// Timeout is the per-attempt request timeout. This can be configured using
	// WithTimeout().
	Timeout time.Duration

	// AppKey is the TfL Unified API application key to attach to requests in
	// the `app_key` header. If empty, anonymous access will be used. This can
	// be configured using WithAppKey().
	AppKey string

	req *http.Request
}

// ClientOption allows customising the client's behaviour during construction
// with NewClient().
type ClientOption func(*Client)

// WithAppKey configures an application key to attach to API calls. This can be
// obtained from https://api-portal.tfl.gov.uk/, and increases the request
// limit from 50 to 500 calls per minute.
func WithAppKey(key string) ClientOption {
	return func(c *Client) {
		c.AppKey = key
	}
}

// WithTimeout sets the per-attempt request timeout. This is 3s by default. The
// lower of this and the HTTPClient's request timeout will be effective.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.Timeout = timeout
	}
}

// NewClient initialises a client to retrieve data from the BikePoint API.
func NewClient(logger *slog.Logger, httpClient *http.Client, opts ...ClientOption) *Client {
	c := &Client{
		Logger:     logger,
		HTTPClient: httpClient,
		Timeout:    3 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.req = c.buildRequest()
	return c
}

func (c Client) buildRequest() *http.Request {
	req, err := http.NewRequest(http.MethodGet, "https://api.tfl.gov.uk/BikePoint", nil)
	if err != nil {
		// We're fully in control of this part; the tests will fail if this
		// returns an error.
		panic(err)
	}

	// TfL blocks the default "Go-http-client/1.1" user agent.
	req.Header.Set("user-agent", "tflcycles_exporter/"+stamp.Version)

	if c.AppKey != "" {
		// This is not documented, but is supported:
		// https://techforum.tfl.gov.uk/t/use-of-http-header-for-app-key/3113
		req.Header.Set("app_key", c.AppKey)
	}

	return req
}

// FetchStationAvailabilities retrieves the latest cycle and dock availability.
// It will back-off exponentially until the passed context expires. The
// returned list can be assumed to be sorted by station ID.
func (c *Client) FetchStationAvailabilities(ctx context.Context) ([]StationAvailability, error) {
	// Can still grow if needed; this saves the first handful of reallocs.
	stationAvailabilities := make([]StationAvailability, 0, 1024)
	err := backoff.RetryNotify(
		func() error {
			ctx, cancel := context.WithTimeout(ctx, c.Timeout)
			defer cancel()

			timer := prometheus.NewTimer(httpRequestDuration)
			defer timer.ObserveDuration()

			resp, err := c.HTTPClient.Do(c.req.WithContext(ctx))
			if err != nil {
				httpRequestFailures.Inc()
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				httpRequestFailures.Inc()
				msg := fmt.Sprintf("got HTTP %v", resp.StatusCode)
				var fault error
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					fault = errors.New(msg)
				} else {
					fault = fmt.Errorf("%v: %v", msg, string(b))
				}
				if resp.StatusCode < http.StatusInternalServerError {
					return backoff.Permanent(fault)
				}
				return fault
			}

			dec := json.NewDecoder(resp.Body)
			if err := dec.Decode(&stationAvailabilities); err != nil {
				httpRequestFailures.Inc()
				// In case we partially decoded the response.
				stationAvailabilities = stationAvailabilities[:0]
				return err
			}
			return nil
		},
		// No need to use backoff.WithContext() here as well.
		backoff.NewExponentialBackOff(),
		func(err error, wait time.Duration) {
			c.Logger.WarnContext(ctx, "failed attempt",
				slog.String("error", err.Error()),
				// This may not be relevant to the error above, but typically
				// it is.
				slog.Duration("timeout", c.Timeout),
				slog.Duration("wait", wait))
			httpRequestRetries.Inc()
		},
	)
	return stationAvailabilities, err
}
