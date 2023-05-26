package exporter

import (
	"github.com/gebn/tflcycles_exporter/internal/pkg/tflcycles"

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
	bikesAvailable = prometheus.NewDesc(
		"tflcycles_bikes_available",
		"The number of in-service, conventional bikes available for hire.",
		[]string{"station"},
		nil,
	)
	ebikesAvailable = prometheus.NewDesc(
		"tflcycles_ebikes_available",
		"The number of in-service e-bikes available for hire.",
		[]string{"station"},
		nil,
	)
)

type StationAvailabilitiesCollector struct {
	StationAvailabilities []tflcycles.StationAvailability
}

func (StationAvailabilitiesCollector) Describe(d chan<- *prometheus.Desc) {
	d <- docks
	d <- docksAvailable
	d <- bikesAvailable
	d <- ebikesAvailable
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
			bikesAvailable,
			prometheus.GaugeValue,
			float64(stationAvailability.Availability.Bikes),
			stationAvailability.Station.Name,
		)
		m <- prometheus.MustNewConstMetric(
			ebikesAvailable,
			prometheus.GaugeValue,
			float64(stationAvailability.Availability.EBikes),
			stationAvailability.Station.Name,
		)
	}
}
