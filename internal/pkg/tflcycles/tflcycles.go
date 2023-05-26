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
	//ID int // e.g. 112
	Name string // e.g. Stonecutter Street, Holborn
	// Terminal string // e.g. 001061
	// Latitude float32 // lat, WGS84 latitude of the location
	// Longitude float32 // lon, WGS84 longitude of the location
	// Installed time.Time // InstallDate, zero value means not installed // This will not work as same stations are installed but do not have an install date
	// IsTemporary bool
	// IsLocked bool // what does this mean - whole thing out of action?

	Docks int // NbDocks
}

type Availability struct {
	// This was originally called Occupancy, however that made a Docks field
	// ambiguous. Now, values in this struct are all what is available to use.

	// AsOf time.Time // modified
	Docks  int // NbEmptyDocks
	Bikes  int // NbStandardBikes
	EBikes int // NbEBikes
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
