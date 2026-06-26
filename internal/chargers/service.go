package chargers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tamcore/voltpilot/internal/cache"
	"github.com/tamcore/voltpilot/internal/enbw"
	"github.com/tamcore/voltpilot/internal/geo"
)

// ErrNotFound is returned when a station id has no detail.
var ErrNotFound = errors.New("charger not found")

const (
	defaultRadiusKm = 10.0
	maxRadiusKm     = 100.0
	defaultLimit    = 25
	maxLimit        = 200
	cacheTTL        = 45 * time.Second

	// The EnBW API clusters dense areas even with grouping=false: a wide bbox
	// returns "grouped" items with a null stationId and a viewPort instead of
	// the individual stations. We resolve those by re-querying each cluster's
	// viewPort (nearest-first, bounded, in parallel) so nearby stations are
	// never hidden behind a cluster.
	maxClusterExpansions = 24
	expandConcurrency    = 6
	expandDepth          = 2
)

// stationLister is the slice of the EnBW client this service needs.
type stationLister interface {
	List(ctx context.Context, b geo.BBox, grouping bool) ([]enbw.Station, error)
	Detail(ctx context.Context, id int) (*enbw.StationDetail, error)
}

// Service builds the typed charger views over the EnBW client, with a short
// TTL cache on bounding-box list queries.
type Service struct {
	client stationLister
	cache  *cache.TTL[[]enbw.Station]
}

// NewService constructs a Service.
func NewService(client stationLister) *Service {
	return &Service{client: client, cache: cache.NewTTL[[]enbw.Station](cacheTTL)}
}

// Query parameterizes a nearby-chargers lookup.
type Query struct {
	Center        geo.LatLng
	RadiusKm      float64
	OperatorCode  string
	Current       string // "ac" | "dc" | "all"
	AvailableOnly bool
	Limit         int
}

// Nearby returns this CPO's chargers near the center, filtered and ranked by
// distance ascending.
func (s *Service) Nearby(ctx context.Context, q Query) ([]Charger, error) {
	radius := clampRadius(q.RadiusKm)
	stations, err := s.listCached(ctx, q.Center, radius)
	if err != nil {
		return nil, err
	}

	operator := strings.TrimSpace(q.OperatorCode)
	out := make([]Charger, 0, len(stations))
	for i := range stations {
		st := stations[i]
		if st.Grouped || st.StationID == nil {
			continue // skip clusters; we query grouping=false anyway
		}
		if operator != "" && !strings.EqualFold(st.OperatorCode, operator) {
			continue
		}
		cur := classifyCurrent(st.PlugTypes, st.MaxPowerInKw)
		if !matchesCurrent(cur, q.Current) {
			continue
		}
		available := st.AvailableChargePoints > 0
		if q.AvailableOnly && !available {
			continue
		}
		out = append(out, toCharger(st, cur, available, q.Center))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })

	limit := clampLimit(q.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CPOsNearby returns the distinct operators present near the center, ordered
// by how many stations each has (most first).
func (s *Service) CPOsNearby(ctx context.Context, center geo.LatLng, radiusKm float64) ([]CPO, error) {
	stations, err := s.listCached(ctx, center, clampRadius(radiusKm))
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]*CPO)
	for i := range stations {
		st := stations[i]
		if st.Grouped || st.OperatorCode == "" {
			continue
		}
		c, ok := byCode[st.OperatorCode]
		if !ok {
			c = &CPO{OperatorCode: st.OperatorCode, Operator: st.Operator}
			byCode[st.OperatorCode] = c
		}
		c.Count++
	}
	out := make([]CPO, 0, len(byCode))
	for _, c := range byCode {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Operator < out[j].Operator
	})
	return out, nil
}

// ChargerDetail fetches one station's detail and maps it to the detail view.
func (s *Service) ChargerDetail(ctx context.Context, id string, center geo.LatLng) (*Detail, error) {
	num, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid id", ErrNotFound)
	}
	d, err := s.client.Detail(ctx, num)
	if err != nil {
		return nil, err
	}
	if d == nil || d.StationID == nil {
		return nil, ErrNotFound
	}

	cur := classifyCurrent(d.PlugTypes, d.MaxPowerInKw)
	available := d.AvailableChargePoints > 0
	base := toCharger(d.Station, cur, available, center)

	points := make([]ChargePoint, 0, len(d.ChargePoints))
	for _, cp := range d.ChargePoints {
		conns := make([]Connector, 0, len(cp.Connectors))
		for _, cn := range cp.Connectors {
			tg := ""
			if cn.TariffInfo != nil {
				tg = cn.TariffInfo.TariffGroup
			}
			conns = append(conns, Connector{
				PlugTypeGroup: cn.ChargePlugTypeGroup,
				PlugTypeName:  cn.PlugTypeName,
				MaxPowerKw:    cn.MaxPowerInKw,
				Current:       classifyConnector(tg, cn.ChargePlugTypeGroup, cn.MaxPowerInKw),
				CableAttached: cn.CableAttached,
			})
		}
		points = append(points, ChargePoint{
			EvseID:     cp.EvseID,
			Status:     cp.Status,
			Available:  strings.EqualFold(cp.Status, "AVAILABLE"),
			Connectors: conns,
		})
	}

	return &Detail{Charger: base, StationSummary: d.StationSummary, ChargePoints: points}, nil
}

func (s *Service) listCached(ctx context.Context, center geo.LatLng, radiusKm float64) ([]enbw.Station, error) {
	b := geo.BBoxAround(center, radiusKm)
	key := fmt.Sprintf("%.4f,%.4f,%.4f,%.4f", b.MinLat, b.MinLon, b.MaxLat, b.MaxLon)
	if v, ok := s.cache.Get(key); ok {
		return v, nil
	}
	stations, err := s.gather(ctx, b, center)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, stations)
	return stations, nil
}

// gather fetches the bbox and resolves any clusters into individual stations
// by expanding their viewPorts (nearest-first, bounded depth + count). It
// returns deduplicated individual stations only.
func (s *Service) gather(ctx context.Context, root geo.BBox, center geo.LatLng) ([]enbw.Station, error) {
	first, err := s.client.List(ctx, root, false)
	if err != nil {
		return nil, err
	}

	byID := make(map[int]enbw.Station)
	var clusters []enbw.Station
	collect := func(sts []enbw.Station) {
		for _, st := range sts {
			if st.Grouped {
				if st.ViewPort != nil {
					clusters = append(clusters, st)
				}
				continue
			}
			if st.StationID != nil {
				if _, ok := byID[*st.StationID]; !ok {
					byID[*st.StationID] = st
				}
			}
		}
	}
	collect(first)

	left := maxClusterExpansions
	for depth := 0; depth < expandDepth && len(clusters) > 0 && left > 0; depth++ {
		sort.Slice(clusters, func(i, j int) bool {
			di := geo.Distance(center, geo.LatLng{Lat: clusters[i].Lat, Lon: clusters[i].Lon})
			dj := geo.Distance(center, geo.LatLng{Lat: clusters[j].Lat, Lon: clusters[j].Lon})
			return di < dj
		})
		n := len(clusters)
		if n > left {
			n = left
		}
		batch := clusters[:n]
		left -= n
		clusters = nil // next-level clusters are re-collected from the results

		for _, sts := range s.expandClusters(ctx, batch) {
			collect(sts)
		}
	}

	out := make([]enbw.Station, 0, len(byID))
	for _, st := range byID {
		out = append(out, st)
	}
	return out, nil
}

// expandClusters re-queries each cluster's viewPort in parallel (bounded
// concurrency) and returns the per-cluster station lists. Failed expansions
// yield nil and are simply skipped.
func (s *Service) expandClusters(ctx context.Context, clusters []enbw.Station) [][]enbw.Station {
	out := make([][]enbw.Station, len(clusters))
	sem := make(chan struct{}, expandConcurrency)
	var wg sync.WaitGroup
	for i := range clusters {
		vp := clusters[i].ViewPort
		if vp == nil {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, vp enbw.ViewPort) {
			defer wg.Done()
			defer func() { <-sem }()
			b := geo.BBox{
				MinLat: vp.LowerLeftLat,
				MinLon: vp.LowerLeftLon,
				MaxLat: vp.UpperRightLat,
				MaxLon: vp.UpperRightLon,
			}
			if sts, err := s.client.List(ctx, b, false); err == nil {
				out[idx] = sts
			}
		}(i, *vp)
	}
	wg.Wait()
	return out
}

func toCharger(st enbw.Station, cur Current, available bool, center geo.LatLng) Charger {
	pos := geo.LatLng{Lat: st.Lat, Lon: st.Lon}
	id := ""
	if st.StationID != nil {
		id = strconv.Itoa(*st.StationID)
	}
	addr := ""
	if st.ShortAddress != nil {
		addr = *st.ShortAddress
	}
	return Charger{
		ID:                    id,
		Operator:              st.Operator,
		OperatorCode:          st.OperatorCode,
		Lat:                   st.Lat,
		Lon:                   st.Lon,
		DistanceKm:            round1(geo.Distance(center, pos) / 1000),
		Address:               addr,
		MaxPowerKw:            st.MaxPowerInKw,
		PlugTypes:             st.PlugTypes,
		PlugTypeNames:         st.PlugTypeNames,
		Current:               cur,
		NumberOfChargePoints:  st.NumberOfChargePoints,
		AvailableChargePoints: st.AvailableChargePoints,
		Available:             available,
		AlwaysOpen:            st.AlwaysOpen,
		DeepLinks:             deepLinks(pos),
	}
}

func deepLinks(p geo.LatLng) DeepLinks {
	return DeepLinks{
		Google: fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%g,%g&travelmode=driving", p.Lat, p.Lon),
		Apple:  fmt.Sprintf("https://maps.apple.com/?daddr=%g,%g&dirflg=d", p.Lat, p.Lon),
		Waze:   fmt.Sprintf("https://waze.com/ul?ll=%g,%g&navigate=yes", p.Lat, p.Lon),
	}
}

func round1(v float64) float64 { return float64(int64(v*10+0.5)) / 10 }

func clampRadius(r float64) float64 {
	if r <= 0 {
		return defaultRadiusKm
	}
	if r > maxRadiusKm {
		return maxRadiusKm
	}
	return r
}

func clampLimit(l int) int {
	if l <= 0 {
		return defaultLimit
	}
	if l > maxLimit {
		return maxLimit
	}
	return l
}
