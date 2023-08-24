package tflcycles

import (
	"encoding/json"
	"regexp"
	"strconv"
)

var (
	// propertyMappings provides an efficient way to take a given
	// additionalProperty in the response and find the corresponding field on a
	// StationAvailability struct to set with its value.
	propertyMappings = map[string]func(*StationAvailability) *int{
		"NbEmptyDocks": func(sa *StationAvailability) *int {
			return &sa.Availability.Docks
		},
		"NbDocks": func(sa *StationAvailability) *int {
			return &sa.Station.Docks
		},
		"NbStandardBikes": func(sa *StationAvailability) *int {
			return &sa.Availability.Bicycles
		},
		"NbEBikes": func(sa *StationAvailability) *int {
			return &sa.Availability.EBikes
		},
	}
)

// Station contains relatively-stable metadata about a docking point.
type Station struct {

	// Name is the human-readable location of the docking point, e.g.
	// "Stonecutter Street, Holborn". It is taken from the `commonName` field
	// of the JSON.
	Name string

	// Docks indicates the total number of docks at the docking point,
	// including those out of service. It is taken from the `NbDocks` property
	// of the JSON.
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

	// Bicycles is the number of in-service, non-electric bikes available for
	// hire. It is taken from the `NbStandardBikes` property.
	Bicycles int

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

var (
	whitespaceBeforeComma = regexp.MustCompile(`\s+,`)
)

// normaliseCommonName removes whitespace before the comma in bike point names.
// 39 stations currently have one space before the comma, and 1 has two
// ("Kennington Road  , Vauxhall").
func normaliseCommonName(commonName string) string {
	return whitespaceBeforeComma.ReplaceAllString(commonName, ",")
}

func (sa *StationAvailability) UnmarshalJSON(b []byte) error {
	p := place{}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}

	sa.Station.Name = normaliseCommonName(p.CommonName)

	for _, ap := range p.AdditionalProperties {
		mapping, ok := propertyMappings[ap.Key]
		if !ok {
			continue
		}
		field := mapping(sa)

		value, err := strconv.Atoi(ap.Value)
		if err != nil {
			return err
		}
		*field = value
	}
	return nil
}
