package tflcycles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Client does not implement station ID filtering as there would be no point -
// we'd have to decode the JSON and then delete bits from the output. We may as
// well leave this to the library user.
type Client struct {
	HTTPClient *http.Client
	AppKey     string

	req *http.Request
}

type ClientOption func(*Client)

func WithAppKey(key string) ClientOption {
	return func(c *Client) {
		c.AppKey = key
	}
}

func NewClient(httpClient *http.Client, opts ...ClientOption) *Client {
	c := &Client{
		HTTPClient: httpClient,
	}
	for _, opt := range opts {
		opt(c)
	}

	c.req = c.buildRequest()
	return c
}

func (c *Client) buildRequest() *http.Request {
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

// Returned list can be assumed to be sorted by station ID.
func (c *Client) FetchStationAvailabilities(ctx context.Context) ([]StationAvailability, error) {
	resp, err := c.HTTPClient.Do(c.req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("got HTTP %v", resp.StatusCode)
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(msg)
		}
		return nil, fmt.Errorf("%v: %v", msg, string(b))
	}

	d := json.NewDecoder(resp.Body)
	// We can still grow if needed; this saves the first handful of reallocs.
	sa := make([]StationAvailability, 0, 1024)
	if err := d.Decode(&sa); err != nil {
		return nil, err
	}
	return sa, nil
}
