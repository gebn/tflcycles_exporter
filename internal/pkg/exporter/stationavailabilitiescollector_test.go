package exporter

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gebn/tflcycles_exporter/internal/pkg/bikepoint"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestStationAvailabilitiesCollector_Collect(t *testing.T) {
	tests := []struct {
		c    StationAvailabilitiesCollector
		want string
	}{
		{
			StationAvailabilitiesCollector{
				[]bikepoint.StationAvailability{
					{
						Station: bikepoint.Station{
							Name:  "Foo",
							Docks: 5,
						},
						Availability: bikepoint.Availability{
							Docks:    1,
							Bicycles: 2,
							EBikes:   1,
						},
					},
					{
						Station: bikepoint.Station{
							Name:  "Bar",
							Docks: 22,
						},
						Availability: bikepoint.Availability{
							Docks:    1,
							Bicycles: 3,
							EBikes:   5,
						},
					},
				},
			},
			`
			# HELP tflcycles_bicycles_available The number of in-service, conventional bikes available for hire.
            # TYPE tflcycles_bicycles_available gauge
            tflcycles_bicycles_available{station="Bar"} 3
            tflcycles_bicycles_available{station="Foo"} 2
            # HELP tflcycles_docks The total number of docks at the station, including those that are out of service.
            # TYPE tflcycles_docks gauge
            tflcycles_docks{station="Bar"} 22
            tflcycles_docks{station="Foo"} 5
            # HELP tflcycles_docks_available The number of in-service, vacant docks to which a bike can be returned.
            # TYPE tflcycles_docks_available gauge
            tflcycles_docks_available{station="Bar"} 1
            tflcycles_docks_available{station="Foo"} 1
            # HELP tflcycles_ebikes_available The number of in-service e-bikes available for hire.
            # TYPE tflcycles_ebikes_available gauge
            tflcycles_ebikes_available{station="Bar"} 5
            tflcycles_ebikes_available{station="Foo"} 1
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
