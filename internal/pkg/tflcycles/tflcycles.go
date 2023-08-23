package tflcycles

import (
	"encoding/json"
	"regexp"
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

	whitespaceBeforeComma = regexp.MustCompile(`\s+,`)
)

func init() {
	relevantPropertiesLookup = make(map[string]struct{}, len(relevantProperties))
	for _, p := range relevantProperties {
		relevantPropertiesLookup[p] = struct{}{}
	}
}

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
		if _, ok := relevantPropertiesLookup[ap.Key]; !ok {
			continue
		}
		// All relevantProperties are ints.
		v, err := strconv.Atoi(ap.Value)
		if err != nil {
			return err
		}
		switch ap.Key {
		case "NbDocks":
			sa.Station.Docks = v
		case "NbEmptyDocks":
			sa.Availability.Docks = v
		case "NbStandardBikes":
			sa.Availability.Bicycles = v
		case "NbEBikes":
			sa.Availability.EBikes = v
		}
	}
	return nil
}
