package exporter

import (
	"github.com/gebn/tflcycles_exporter/internal/pkg/bikepoint"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	docks = prometheus.NewDesc(
		"tflcycles_docks",
		"The total number of docks at the station, including those that are out of service.",
		[]string{"station"},
		nil,
	)
	docksAvailable = prometheus.NewDesc(
		"tflcycles_docks_available",
		"The number of in-service, vacant docks to which a bike can be returned.",
		[]string{"station"},
		nil,
	)
	bicyclesAvailable = prometheus.NewDesc(
		"tflcycles_bicycles_available",
		"The number of in-service, conventional bikes available for hire.",
		[]string{"station"},
		nil,
	)
	eBikesAvailable = prometheus.NewDesc(
		"tflcycles_ebikes_available",
		"The number of in-service e-bikes available for hire.",
		[]string{"station"},
		nil,
	)
)

// StationAvailabilitiesCollector is a prometheus.Collector yielding metrics
// about retrieved dock and bike availability data.
type StationAvailabilitiesCollector struct {
	StationAvailabilities []bikepoint.StationAvailability
}

func (StationAvailabilitiesCollector) Describe(d chan<- *prometheus.Desc) {
	d <- docks
	d <- docksAvailable
	d <- bicyclesAvailable
	d <- eBikesAvailable
}

func (c StationAvailabilitiesCollector) Collect(m chan<- prometheus.Metric) {
	for _, stationAvailability := range c.StationAvailabilities {
		m <- prometheus.MustNewConstMetric(
			docks,
			prometheus.GaugeValue,
			float64(stationAvailability.Station.Docks),
			stationAvailability.Station.Name,
		)
		m <- prometheus.MustNewConstMetric(
			docksAvailable,
			prometheus.GaugeValue,
			float64(stationAvailability.Availability.Docks),
			stationAvailability.Station.Name,
		)
		m <- prometheus.MustNewConstMetric(
			bicyclesAvailable,
			prometheus.GaugeValue,
			float64(stationAvailability.Availability.Bicycles),
			stationAvailability.Station.Name,
		)
		m <- prometheus.MustNewConstMetric(
			eBikesAvailable,
			prometheus.GaugeValue,
			float64(stationAvailability.Availability.EBikes),
			stationAvailability.Station.Name,
		)
	}
}
