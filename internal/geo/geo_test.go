package geo

import (
	"math"
	"testing"
)

func TestDistanceZero(t *testing.T) {
	a := LatLng{Lat: 49.778, Lon: 10.066}
	if d := Distance(a, a); d != 0 {
		t.Fatalf("distance to self = %v, want 0", d)
	}
}

func TestDistanceKnown(t *testing.T) {
	// ~1 deg of latitude ≈ 111 km.
	a := LatLng{Lat: 49.0, Lon: 10.0}
	b := LatLng{Lat: 50.0, Lon: 10.0}
	got := Distance(a, b)
	want := 111195.0 // meters, great-circle
	if math.Abs(got-want) > 500 {
		t.Fatalf("distance = %v, want ~%v", got, want)
	}
}

func TestBBoxAroundContainsCenter(t *testing.T) {
	c := LatLng{Lat: 49.778, Lon: 10.066}
	b := BBoxAround(c, 10)
	if c.Lat <= b.MinLat || c.Lat >= b.MaxLat {
		t.Fatalf("center lat %v not inside [%v,%v]", c.Lat, b.MinLat, b.MaxLat)
	}
	if c.Lon <= b.MinLon || c.Lon >= b.MaxLon {
		t.Fatalf("center lon %v not inside [%v,%v]", c.Lon, b.MinLon, b.MaxLon)
	}
}

func TestBBoxAroundRadiusRoughlyCorrect(t *testing.T) {
	c := LatLng{Lat: 49.778, Lon: 10.066}
	b := BBoxAround(c, 10)
	// The north edge should be ~10 km from the center.
	north := LatLng{Lat: b.MaxLat, Lon: c.Lon}
	d := Distance(c, north) / 1000
	if math.Abs(d-10) > 0.5 {
		t.Fatalf("north edge distance = %v km, want ~10", d)
	}
}

func TestBBoxAroundNonPositiveRadius(t *testing.T) {
	c := LatLng{Lat: 0, Lon: 0}
	b := BBoxAround(c, 0)
	if b.MaxLat <= b.MinLat {
		t.Fatal("expected a non-degenerate bbox for radius<=0 (defaults to 1km)")
	}
}
