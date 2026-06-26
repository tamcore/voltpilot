package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/voltpilot/internal/chargers"
	"github.com/tamcore/voltpilot/internal/geo"
)

type fakeSvc struct {
	chargers []chargers.Charger
	cpos     []chargers.CPO
	detail   *chargers.Detail
	err      error
}

func (f *fakeSvc) Nearby(context.Context, chargers.Query) ([]chargers.Charger, error) {
	return f.chargers, f.err
}
func (f *fakeSvc) CPOsNearby(context.Context, geo.LatLng, float64) ([]chargers.CPO, error) {
	return f.cpos, f.err
}
func (f *fakeSvc) ChargerDetail(context.Context, string, geo.LatLng) (*chargers.Detail, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.detail, nil
}

func newRouter(svc Service) http.Handler {
	r := chi.NewRouter()
	NewChargers(svc).Mount(r)
	return r
}

func TestListRequiresLatLon(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chargers", nil)
	newRouter(&fakeSvc{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestListReturnsChargers(t *testing.T) {
	svc := &fakeSvc{chargers: []chargers.Charger{{ID: "1", OperatorCode: "DEBPE"}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chargers?lat=49.778&lon=10.066&operatorCode=DEBPE", nil)
	newRouter(svc).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var body struct {
		Chargers []chargers.Charger `json:"chargers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Chargers) != 1 || body.Chargers[0].ID != "1" {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestCPOsEndpoint(t *testing.T) {
	svc := &fakeSvc{cpos: []chargers.CPO{{OperatorCode: "DEBPE", Operator: "Aral pulse", Count: 3}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cpos?lat=49.778&lon=10.066", nil)
	newRouter(svc).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestDetailNotFound(t *testing.T) {
	svc := &fakeSvc{err: chargers.ErrNotFound}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chargers/999?lat=49.778&lon=10.066", nil)
	newRouter(svc).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}
