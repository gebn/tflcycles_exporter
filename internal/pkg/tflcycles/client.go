package tflcycles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

// Client does not implement station ID filtering as there would be no point -
// we'd have to decode the JSON and then delete bits from the output. We may as
// well leave this to the library user.
type Client struct {
	HTTPClient *http.Client
	Timeout    time.Duration
	AppKey     string

	req *http.Request
}

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

func NewClient(httpClient *http.Client, opts ...ClientOption) *Client {
	c := &Client{
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

	// Otherwise we may receive data up to 30s old.
	req.Header.Set("cache-control", "no-cache")

	// TfL blocks the default "Go-http-client/1.1" user agent.
	req.Header.Set("user-agent", "tflcycles_exporter/v0.0.1") // TODO Use actual version.

	if c.AppKey != "" {
		q := req.URL.Query()
		q.Set("app_key", c.AppKey)
		req.URL.RawQuery = q.Encode()
	}

	return req
}

// FetchStationAvailabilities retrieves the latest cycle and dock availability.
// The returned list can be assumed to be sorted by station ID.
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
			log.Printf("failed attempt: %v", err)
			httpRequestRetries.Inc()
		},
	)
	return stationAvailabilities, err
}
