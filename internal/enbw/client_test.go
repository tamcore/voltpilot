package enbw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/tamcore/voltpilot/internal/geo"
)

type stubKeys struct {
	mu        sync.Mutex
	key       string
	refreshed int
}

func (s *stubKeys) Key() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.key
}
func (s *stubKeys) Refresh(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshed++
	s.key = "fresh-key"
	return nil
}

func TestClientListAndDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(subKeyHeader) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/chargestations" {
			_, _ = w.Write([]byte(`[{"grouped":false,"stationId":1,"operator":"EnBW","operatorCode":"DEEBW","lat":49.8,"lon":9.9,"plugTypes":["CCS"],"maxPowerInKw":300,"numberOfChargePoints":2,"availableChargePoints":1}]`))
			return
		}
		_, _ = w.Write([]byte(`{"grouped":false,"stationId":42,"operator":"EnBW","operatorCode":"DEEBW","lat":49.8,"lon":9.9,"chargePoints":[{"evseId":"DE*X*1","status":"AVAILABLE","connectors":[{"chargePlugTypeGroup":"CCS","plugTypeName":"CCS","maxPowerInKw":300,"tariffInfo":{"tariffGroup":"DC_CHARGER"}}]}]}`))
	}))
	defer srv.Close()

	c := NewClient(&stubKeys{key: "k"}, srv.URL, srv.Client())

	stations, err := c.List(context.Background(), geo.BBox{MinLat: 49, MinLon: 9, MaxLat: 50, MaxLon: 10}, false)
	if err != nil || len(stations) != 1 || stations[0].OperatorCode != "DEEBW" {
		t.Fatalf("List failed: %v %+v", err, stations)
	}

	d, err := c.Detail(context.Background(), 42)
	if err != nil || d.StationID == nil || *d.StationID != 42 || len(d.ChargePoints) != 1 {
		t.Fatalf("Detail failed: %v %+v", err, d)
	}
	if d.ChargePoints[0].Connectors[0].TariffInfo.TariffGroup != "DC_CHARGER" {
		t.Fatalf("connector tariff not decoded: %+v", d.ChargePoints[0])
	}
}

func TestClientRefreshesKeyOn403(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusForbidden) // throttle/expired key
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	keys := &stubKeys{key: "stale"}
	c := NewClient(keys, srv.URL, srv.Client())
	if _, err := c.List(context.Background(), geo.BBox{}, false); err != nil {
		t.Fatalf("expected success after key refresh, got %v", err)
	}
	if keys.refreshed != 1 {
		t.Fatalf("expected exactly one key refresh, got %d", keys.refreshed)
	}
	if calls != 2 {
		t.Fatalf("expected one retry (2 calls), got %d", calls)
	}
}

func TestClientNoKeyErrors(t *testing.T) {
	// Key empty and Refresh leaves it empty → request must fail fast.
	c := NewClient(&emptyKeys{}, "http://127.0.0.1:0", nil)
	if _, err := c.List(context.Background(), geo.BBox{}, false); err == nil {
		t.Fatal("expected error when no key is available")
	}
}

type emptyKeys struct{}

func (emptyKeys) Key() string                   { return "" }
func (emptyKeys) Refresh(context.Context) error { return nil }

func TestKeyManagerRefreshScrapes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<script>var c={apimSubscriptionKey: "0123456789abcdef0123456789abcdef"};</script>`))
	}))
	defer srv.Close()

	m := NewKeyManager(srv.URL, "", srv.Client())
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if m.Key() != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("scraped key = %q", m.Key())
	}
}

func TestKeyManagerRefreshKeepsKeyOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	m := NewKeyManager(srv.URL, "seed", srv.Client())
	if err := m.Refresh(context.Background()); err == nil {
		t.Fatal("expected error on 500")
	}
	if m.Key() != "seed" {
		t.Fatalf("seed key must survive a failed refresh, got %q", m.Key())
	}
}
