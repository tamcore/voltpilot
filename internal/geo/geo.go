// Package geo provides geographic primitives: coordinates, great-circle
// distance, and bounding-box derivation for the EnBW spatial queries.
package geo

import "math"

// earthRadiusMeters is the WGS84 mean radius used by Distance.
const earthRadiusMeters = 6371008.8

// LatLng is a WGS84 coordinate pair in decimal degrees.
type LatLng struct {
	Lat float64
	Lon float64
}

// BBox is an axis-aligned geographic bounding box.
type BBox struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

// Distance returns the great-circle distance between two coordinates in meters.
func Distance(a, b LatLng) float64 {
	phi1 := degToRad(a.Lat)
	phi2 := degToRad(b.Lat)
	dPhi := degToRad(b.Lat - a.Lat)
	dLam := degToRad(b.Lon - a.Lon)

	h := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLam/2)*math.Sin(dLam/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// BBoxAround returns a bounding box centred on c that extends radiusKm in every
// direction. Longitude degrees are scaled by cos(lat) so the box stays roughly
// square in real distance regardless of latitude.
func BBoxAround(c LatLng, radiusKm float64) BBox {
	if radiusKm <= 0 {
		radiusKm = 1
	}
	const kmPerDegLat = 111.32
	dLat := radiusKm / kmPerDegLat
	cosLat := math.Cos(degToRad(c.Lat))
	if cosLat < 0.01 {
		cosLat = 0.01 // guard against the poles
	}
	dLon := radiusKm / (kmPerDegLat * cosLat)
	return BBox{
		MinLat: c.Lat - dLat,
		MinLon: c.Lon - dLon,
		MaxLat: c.Lat + dLat,
		MaxLon: c.Lon + dLon,
	}
}

func degToRad(d float64) float64 { return d * math.Pi / 180 }
