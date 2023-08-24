package exporter

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestScrapeCollector_Collect(t *testing.T) {
	tests := []struct {
		c    ScrapeCollector
		want string
	}{
		{
			ScrapeCollector{true, 2 * time.Second},
			`
            # HELP tflcycles_scrape_duration_seconds The amount of time it took to retrieve and parse the data for the scrape.
            # TYPE tflcycles_scrape_duration_seconds gauge
            tflcycles_scrape_duration_seconds 2
            # HELP tflcycles_up Whether the request to TfL's /BikePoint API succeeded.
            # TYPE tflcyles_up untyped
            tflcycles_up 1
            `,
		},
		{
			ScrapeCollector{false, time.Second},
			`
            # HELP tflcycles_scrape_duration_seconds The amount of time it took to retrieve and parse the data for the scrape.
            # TYPE tflcycles_scrape_duration_seconds gauge
            tflcycles_scrape_duration_seconds 1
            # HELP tflcycles_up Whether the request to TfL's /BikePoint API succeeded.
            # TYPE tflcyles_up untyped
            tflcycles_up 0
            `,
		},
	}
	for i, test := range tests {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			if err := testutil.CollectAndCompare(test.c, strings.NewReader(test.want)); err != nil {
				t.Error(err)
			}
		})
	}
}
