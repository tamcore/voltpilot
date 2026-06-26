// Package enbw is a client for the EnBW e-mobility public API
// (enbw-emp.azure-api.net), a roaming aggregator of EV charge stations from
// many Charge Point Operators (CPOs).
package enbw

// Station is one item from the /chargestations bounding-box list, and the
// embedded base of a detail response. When the list is fetched with
// grouping=true, clusters have Grouped=true and a nil StationID.
type Station struct {
	Grouped                  bool     `json:"grouped"`
	StationID                *int     `json:"stationId"`
	Operator                 string   `json:"operator"`
	OperatorCode             string   `json:"operatorCode"`
	ShortAddress             *string  `json:"shortAddress"`
	Lat                      float64  `json:"lat"`
	Lon                      float64  `json:"lon"`
	NumberOfChargePoints     int      `json:"numberOfChargePoints"`
	AvailableChargePoints    int      `json:"availableChargePoints"`
	UnknownStateChargePoints int      `json:"unknownStateChargePoints"`
	PlugTypes                []string `json:"plugTypes"`
	PlugTypeNames            []string `json:"plugTypeNames"`
	MaxPowerInKw             float64   `json:"maxPowerInKw"`
	AlwaysOpen               bool      `json:"alwaysOpen"`
	ViewPort                 *ViewPort `json:"viewPort"`
}

// ViewPort is the bounding box a clustered list item covers. Re-querying it
// with grouping=false resolves the cluster into individual stations.
type ViewPort struct {
	LowerLeftLat  float64 `json:"lowerLeftLat"`
	LowerLeftLon  float64 `json:"lowerLeftLon"`
	UpperRightLat float64 `json:"upperRightLat"`
	UpperRightLon float64 `json:"upperRightLon"`
}

// StationDetail is the /chargestations/{id} response: a Station plus per-EVSE
// charge points and pricing.
type StationDetail struct {
	Station
	StationSummary string        `json:"stationSummary"`
	ChargePoints   []ChargePoint `json:"chargePoints"`
	PricingInfo    *PricingInfo  `json:"pricingInfo"`
}

// ChargePoint is a single EVSE with its live status and connectors.
type ChargePoint struct {
	EvseID     string      `json:"evseId"`
	Status     string      `json:"status"` // AVAILABLE | OCCUPIED | OUT_OF_SERVICE | UNKNOWN
	Connectors []Connector `json:"connectors"`
}

// Connector is one physical plug on a charge point.
type Connector struct {
	ChargePlugTypeGroup string      `json:"chargePlugTypeGroup"` // CCS | CHADEMO | TYPE_2
	PlugTypeName        string      `json:"plugTypeName"`
	MaxPowerInKw        float64     `json:"maxPowerInKw"`
	CableAttached       bool        `json:"cableAttached"`
	TariffInfo          *TariffInfo `json:"tariffInfo"`
}

// TariffInfo carries the AC/DC indicator and pricing text for a connector.
type TariffInfo struct {
	TariffGroup       string `json:"tariffGroup"` // AC_CHARGER | DC_CHARGER
	TariffDescription string `json:"tariffDescription"`
}

// PricingInfo is the station-level pricing summary.
type PricingInfo struct {
	PriceType            string `json:"priceType"`
	ChargePerKwhInCent   int    `json:"chargePerKwhInCent"`
	FreeParkingInMinutes int    `json:"freeParkingInMinutes"`
}
