// Package handlers implements the voltpilot HTTP API handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/voltpilot/internal/chargers"
	"github.com/tamcore/voltpilot/internal/geo"
)

// Service is the charger-querying behaviour the handlers depend on.
type Service interface {
	Nearby(ctx context.Context, q chargers.Query) ([]chargers.Charger, error)
	CPOsNearby(ctx context.Context, center geo.LatLng, radiusKm float64) ([]chargers.CPO, error)
	ChargerDetail(ctx context.Context, id string, center geo.LatLng) (*chargers.Detail, error)
}

// Chargers wires the charger endpoints onto a router.
type Chargers struct {
	svc Service
}

// NewChargers builds the handler set.
func NewChargers(svc Service) *Chargers { return &Chargers{svc: svc} }

// Mount registers routes on r.
func (h *Chargers) Mount(r chi.Router) {
	r.Get("/api/chargers", h.list)
	r.Get("/api/chargers/{id}", h.detail)
	r.Get("/api/cpos", h.cpos)
}

func (h *Chargers) list(w http.ResponseWriter, r *http.Request) {
	center, ok := parseCenter(w, r)
	if !ok {
		return
	}
	q := chargers.Query{
		Center:        center,
		RadiusKm:      parseFloat(r, "radiusKm", 0),
		OperatorCode:  r.URL.Query().Get("operatorCode"),
		Current:       r.URL.Query().Get("current"),
		AvailableOnly: r.URL.Query().Get("availableOnly") == "true",
		Limit:         int(parseFloat(r, "limit", 0)),
	}
	res, err := h.svc.Nearby(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream charge-point service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chargers": res})
}

func (h *Chargers) cpos(w http.ResponseWriter, r *http.Request) {
	center, ok := parseCenter(w, r)
	if !ok {
		return
	}
	res, err := h.svc.CPOsNearby(r.Context(), center, parseFloat(r, "radiusKm", 0))
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream charge-point service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cpos": res})
}

func (h *Chargers) detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	center, ok := parseCenter(w, r)
	if !ok {
		return
	}
	res, err := h.svc.ChargerDetail(r.Context(), id, center)
	if err != nil {
		if errors.Is(err, chargers.ErrNotFound) {
			writeError(w, http.StatusNotFound, "charger not found")
			return
		}
		writeError(w, http.StatusBadGateway, "upstream charge-point service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// parseCenter validates the required lat/lon query params.
func parseCenter(w http.ResponseWriter, r *http.Request) (geo.LatLng, bool) {
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if latErr != nil || lonErr != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		writeError(w, http.StatusBadRequest, "valid lat and lon query params are required")
		return geo.LatLng{}, false
	}
	return geo.LatLng{Lat: lat, Lon: lon}, true
}

func parseFloat(r *http.Request, key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(r.URL.Query().Get(key), 64); err == nil {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
