package tflcycles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gebn/go-stamp/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	// This will observe a shorter value if a given request is terminated early
	// due to timeout.
	httpRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "tflcycles_client_http_request_duration_seconds",
		Help: "Observes the duration of all requests to /BikePoint, including response parsing.",
	})
	httpRequestRetries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tflcycles_client_http_request_retries_total",
		Help: "The number of times we timed-out or received a 5xx error from /BikePoint, and retried.",
	})

	// We do not count failures here, as retries provides an indication of
	// that, and we're more interested in the operation as a whole failing,
	// which is tracked by the exporter.
)

// Client is used to interact with the BikePoint API. Create instances with
// NewClient().
//
// We do not implement station filtering here as there would be no point - we'd
// have to decode the JSON and then delete bits from the output. This is left
// to the user of the client.
type Client struct {

	// Logger will be used to record failed fetch attempts.
	Logger *zap.Logger

	// HTTPClient is the client used to make requests to the API. This must be
	// provided when calling NewClient().
	HTTPClient *http.Client

	// Timeout is the per-attempt request timeout. This can be configured using
	// WithTimeout().
	Timeout time.Duration

	// AppKey is the TfL Unified API application key to attach to requests. If
	// empty, anonymous access will be used. This can be configured using
	// WithAppKey().
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
func NewClient(logger *zap.Logger, httpClient *http.Client, opts ...ClientOption) *Client {
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
		q := req.URL.Query()
		q.Set("app_key", c.AppKey)
		req.URL.RawQuery = q.Encode()
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
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
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
				// In case we've partially decoded the response.
				stationAvailabilities = stationAvailabilities[:0]
				return err
			}
			return nil
		},
		// No need to use backoff.WithContext() here as well.
		backoff.NewExponentialBackOff(),
		func(err error, _ time.Duration) {
			c.Logger.Warn("failed attempt", zap.Error(err))
			httpRequestRetries.Inc()
		},
	)
	return stationAvailabilities, err
}
