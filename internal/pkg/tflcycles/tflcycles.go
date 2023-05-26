package tflcycles

import (
	"encoding/json"
	"strconv"
)

var (
	// relevantProperties holds numeric additionalProperties we care about for
	// decoding.
	relevantProperties = []string{
		"NbEmptyDocks",
		"NbDocks",
		"NbStandardBikes",
		"NbEBikes",
	}

	// relevantPropertiesLookup is used to quickly identify
	// additionalProperties. It is populated once from relevantProperties on
	// package load.
	relevantPropertiesLookup map[string]struct{}
)

func init() {
	relevantPropertiesLookup = make(map[string]struct{}, len(relevantProperties))
	for _, p := range relevantProperties {
		relevantPropertiesLookup[p] = struct{}{}
	}
}

// Station contains relatively-stable metadata about a docking station.
type Station struct {
	// Name is the human-readable location of the station, e.g. "Stonecutter
	// Street, Holborn". It is taken from the `commonName` field of the JSON.
	Name string

	// Docks indicates the total number of docks at the station, including
	// those out of service. It is taken from the `NbDocks` property of the
	// JSON.
	Docks int
}

// Availability represents the hire and drop-off services available at a
// station based on its occupancy.
type Availability struct {
	// This was originally called Occupancy, however that made a Docks field
	// ambiguous. Now, values in this struct all represent resources available
	// to use.

	// Docks is the number of in-service, vacant docks to which a bike can be
	// returned. It is taken from the `NbEmptyDocks` property.
	Docks int

	// Bikes is the number of in-service, non-electric bikes available for
	// hire. It is taken from the `NbStandardBikes` property.
	Bikes int

	// EBikes is the number of in-service, electric bikes available for hire.
	// It is taken from the `NbEBikes` property.
	EBikes int
}

// StationAvailability represents the occupancy of bikes at a particular
// docking station. This could represent the result of a `/BikePoint/{id}`
// query.
type StationAvailability struct {
	Station
	Availability
}

type (
	place struct {
		CommonName           string               `json:"commonName"`
		AdditionalProperties []additionalProperty `json:"additionalProperties"`
	}

	additionalProperty struct {
		Key   string `json:"key"`
		Value string `json:"value"`

		// We do not interpret `modified` as it is too fraught - the Unified
		// API may miss if a bike was rented and returned within the same time
		// interval and not update the timestamp.
	}
)

func (sa *StationAvailability) UnmarshalJSON(b []byte) error {
	// We could decode Name in place, but we wouldn't have
	// additionalProperties.
	var p place
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}

	sa.Station.Name = p.CommonName

	for _, ap := range p.AdditionalProperties {
		if _, ok := relevantPropertiesLookup[ap.Key]; !ok {
			continue
		}
		// All relevantProperties are ints.
		i, err := strconv.Atoi(ap.Value)
		if err != nil {
			return err
		}
		switch ap.Key {
		case "NbDocks":
			sa.Station.Docks = i
		case "NbEmptyDocks":
			sa.Availability.Docks = i
		case "NbStandardBikes":
			sa.Availability.Bikes = i
		case "NbEBikes":
			sa.Availability.EBikes = i
		}
	}
	return nil
}
