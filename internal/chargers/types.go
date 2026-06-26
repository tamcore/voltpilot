// Package chargers turns raw EnBW stations into the typed, filtered, ranked
// shapes the voltpilot frontend consumes.
package chargers

// Current classifies a station's power type.
type Current string

const (
	CurrentAC   Current = "ac"
	CurrentDC   Current = "dc"
	CurrentBoth Current = "both"
)

// DeepLinks are precomputed nav-app handoff URLs for one charger.
type DeepLinks struct {
	Google string `json:"google"`
	Apple  string `json:"apple"`
	Waze   string `json:"waze"`
}

// Charger is one station, ranked by distance, ready for the list view.
type Charger struct {
	ID                    string    `json:"id"`
	Operator              string    `json:"operator"`
	OperatorCode          string    `json:"operatorCode"`
	Lat                   float64   `json:"lat"`
	Lon                   float64   `json:"lon"`
	DistanceKm            float64   `json:"distanceKm"`
	Address               string    `json:"address,omitempty"`
	MaxPowerKw            float64   `json:"maxPowerKw"`
	PlugTypes             []string  `json:"plugTypes"`
	PlugTypeNames         []string  `json:"plugTypeNames"`
	Current               Current   `json:"current"`
	NumberOfChargePoints  int       `json:"numberOfChargePoints"`
	AvailableChargePoints int       `json:"availableChargePoints"`
	Available             bool      `json:"available"`
	AlwaysOpen            bool      `json:"alwaysOpen"`
	DeepLinks             DeepLinks `json:"deep_links"`
}

// CPO is a distinct operator present near a location.
type CPO struct {
	OperatorCode string `json:"operatorCode"`
	Operator     string `json:"operator"`
	Count        int    `json:"count"`
}

// Connector is a per-plug entry on the detail view.
type Connector struct {
	PlugTypeGroup string  `json:"plugTypeGroup"`
	PlugTypeName  string  `json:"plugTypeName"`
	MaxPowerKw    float64 `json:"maxPowerKw"`
	Current       Current `json:"current"`
	CableAttached bool    `json:"cableAttached"`
}

// ChargePoint is one EVSE with live status on the detail view.
type ChargePoint struct {
	EvseID     string      `json:"evseId"`
	Status     string      `json:"status"`
	Available  bool        `json:"available"`
	Connectors []Connector `json:"connectors"`
}

// Detail is the per-charger detail response.
type Detail struct {
	Charger
	StationSummary string        `json:"stationSummary,omitempty"`
	ChargePoints   []ChargePoint `json:"chargePoints"`
}
