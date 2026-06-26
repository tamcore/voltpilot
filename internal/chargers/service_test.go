package chargers

import (
	"context"
	"errors"
	"testing"

	"github.com/tamcore/voltpilot/internal/enbw"
	"github.com/tamcore/voltpilot/internal/geo"
)

// fakeLister is a stub EnBW client.
type fakeLister struct {
	stations []enbw.Station
	detail   *enbw.StationDetail
	listErr  error
	detErr   error
}

func (f *fakeLister) List(_ context.Context, _ geo.BBox, _ bool) ([]enbw.Station, error) {
	return f.stations, f.listErr
}
func (f *fakeLister) Detail(_ context.Context, _ int) (*enbw.StationDetail, error) {
	return f.detail, f.detErr
}

func iptr(i int) *int       { return &i }
func sptr(s string) *string { return &s }
func center() geo.LatLng    { return geo.LatLng{Lat: 49.778, Lon: 10.066} }

func sampleStations() []enbw.Station {
	return []enbw.Station{
		{ // near, DC, available, operator A
			StationID: iptr(1), Operator: "Aral pulse", OperatorCode: "DEBPE",
			Lat: 49.779, Lon: 10.067, PlugTypes: []string{"CCS"}, MaxPowerInKw: 300,
			NumberOfChargePoints: 4, AvailableChargePoints: 2, ShortAddress: sptr("Somewhere 1"),
		},
		{ // far, DC, occupied, operator A
			StationID: iptr(2), Operator: "Aral pulse", OperatorCode: "DEBPE",
			Lat: 49.84, Lon: 10.14, PlugTypes: []string{"CCS"}, MaxPowerInKw: 150,
			NumberOfChargePoints: 2, AvailableChargePoints: 0,
		},
		{ // near, AC, available, operator B
			StationID: iptr(3), Operator: "LichtBlick", OperatorCode: "DEBDO",
			Lat: 49.7795, Lon: 10.0665, PlugTypes: []string{"TYPE_2"}, MaxPowerInKw: 22,
			NumberOfChargePoints: 2, AvailableChargePoints: 1,
		},
		{ // a cluster — must be skipped
			Grouped: true, Operator: "Cluster", OperatorCode: "DEXXX", Lat: 49.78, Lon: 10.07,
		},
	}
}

func TestNearbyFiltersByOperator(t *testing.T) {
	svc := NewService(&fakeLister{stations: sampleStations()})
	got, err := svc.Nearby(context.Background(), Query{Center: center(), OperatorCode: "DEBPE"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d chargers, want 2", len(got))
	}
	for _, c := range got {
		if c.OperatorCode != "DEBPE" {
			t.Fatalf("unexpected operator %q", c.OperatorCode)
		}
	}
}

func TestNearbyRanksByDistance(t *testing.T) {
	svc := NewService(&fakeLister{stations: sampleStations()})
	got, _ := svc.Nearby(context.Background(), Query{Center: center(), OperatorCode: "DEBPE"})
	if got[0].DistanceKm > got[1].DistanceKm {
		t.Fatalf("not sorted ascending: %v then %v", got[0].DistanceKm, got[1].DistanceKm)
	}
	if got[0].ID != "1" {
		t.Fatalf("nearest should be station 1, got %q", got[0].ID)
	}
}

func TestNearbyAvailableOnlyAndCurrent(t *testing.T) {
	svc := NewService(&fakeLister{stations: sampleStations()})
	got, _ := svc.Nearby(context.Background(), Query{
		Center: center(), OperatorCode: "DEBPE", Current: "dc", AvailableOnly: true,
	})
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("available DC filter should yield only station 1, got %+v", got)
	}
	if !got[0].Available {
		t.Fatal("station 1 should be marked available")
	}
	if got[0].DeepLinks.Google == "" {
		t.Fatal("expected google deep link")
	}
}

func TestNearbySkipsClusters(t *testing.T) {
	svc := NewService(&fakeLister{stations: sampleStations()})
	got, _ := svc.Nearby(context.Background(), Query{Center: center()})
	for _, c := range got {
		if c.OperatorCode == "DEXXX" {
			t.Fatal("cluster leaked into results")
		}
	}
}

func TestCPOsNearby(t *testing.T) {
	svc := NewService(&fakeLister{stations: sampleStations()})
	cpos, err := svc.CPOsNearby(context.Background(), center(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(cpos) != 2 {
		t.Fatalf("want 2 distinct CPOs, got %d (%+v)", len(cpos), cpos)
	}
	// DEBPE has 2 stations, should sort first.
	if cpos[0].OperatorCode != "DEBPE" || cpos[0].Count != 2 {
		t.Fatalf("expected DEBPE with count 2 first, got %+v", cpos[0])
	}
}

func TestNearbyListError(t *testing.T) {
	svc := NewService(&fakeLister{listErr: errors.New("boom")})
	if _, err := svc.Nearby(context.Background(), Query{Center: center()}); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestChargerDetail(t *testing.T) {
	det := &enbw.StationDetail{
		Station: enbw.Station{
			StationID: iptr(1888371), Operator: "Aral pulse", OperatorCode: "DEBPE",
			Lat: 49.778, Lon: 10.0657, PlugTypes: []string{"CCS"}, MaxPowerInKw: 300,
			NumberOfChargePoints: 17, AvailableChargePoints: 9,
		},
		ChargePoints: []enbw.ChargePoint{
			{
				EvseID: "DE*BPE*E0F180*01", Status: "AVAILABLE",
				Connectors: []enbw.Connector{
					{ChargePlugTypeGroup: "CCS", PlugTypeName: "CCS (Typ 2)", MaxPowerInKw: 300, CableAttached: true, TariffInfo: &enbw.TariffInfo{TariffGroup: "DC_CHARGER"}},
				},
			},
		},
	}
	svc := NewService(&fakeLister{detail: det})
	got, err := svc.ChargerDetail(context.Background(), "1888371", center())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "1888371" || len(got.ChargePoints) != 1 {
		t.Fatalf("unexpected detail %+v", got)
	}
	if !got.ChargePoints[0].Available || got.ChargePoints[0].Connectors[0].Current != CurrentDC {
		t.Fatalf("connector classification wrong: %+v", got.ChargePoints[0])
	}
}

func TestChargerDetailInvalidID(t *testing.T) {
	svc := NewService(&fakeLister{})
	_, err := svc.ChargerDetail(context.Background(), "not-a-number", center())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
