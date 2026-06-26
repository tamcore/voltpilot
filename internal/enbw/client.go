package enbw

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tamcore/voltpilot/internal/geo"
)

const (
	defaultBaseURL = "https://enbw-emp.azure-api.net/emobility-public-api/api/v1"
	subKeyHeader   = "Ocp-Apim-Subscription-Key"
	requestTimeout = 15 * time.Second

	// browserUserAgent is required: Azure APIM rejects the default Go UA.
	browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:152.0) Gecko/20100101 Firefox/152.0"
)

// keyProvider supplies (and can refresh) the subscription key. *KeyManager
// implements it; tests use a stub.
type keyProvider interface {
	Key() string
	Refresh(ctx context.Context) error
}

// Client talks to the EnBW public charge-station API.
type Client struct {
	baseURL string
	http    *http.Client
	keys    keyProvider
}

// NewClient builds a Client. baseURL may be empty to use the production API.
func NewClient(keys keyProvider, baseURL string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: requestTimeout}
	}
	return &Client{baseURL: baseURL, http: hc, keys: keys}
}

// List fetches charge stations within the bounding box. With grouping=false
// every item is an individual station (StationID populated).
func (c *Client) List(ctx context.Context, b geo.BBox, grouping bool) ([]Station, error) {
	q := url.Values{}
	q.Set("fromLat", ftoa(b.MinLat))
	q.Set("toLat", ftoa(b.MaxLat))
	q.Set("fromLon", ftoa(b.MinLon))
	q.Set("toLon", ftoa(b.MaxLon))
	q.Set("grouping", strconv.FormatBool(grouping))
	q.Set("groupingDivisor", "15")
	u := c.baseURL + "/chargestations?" + q.Encode()

	var stations []Station
	if err := c.getJSON(ctx, u, &stations); err != nil {
		return nil, err
	}
	return stations, nil
}

// Detail fetches one station by id.
func (c *Client) Detail(ctx context.Context, id int) (*StationDetail, error) {
	u := fmt.Sprintf("%s/chargestations/%d", c.baseURL, id)
	var d StationDetail
	if err := c.getJSON(ctx, u, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// getJSON performs an authenticated GET and decodes the JSON body. On a 401/403
// (expired/throttled key) it refreshes the key once and retries.
func (c *Client) getJSON(ctx context.Context, u string, out any) error {
	body, err := c.do(ctx, u)
	if isAuthError(err) {
		if rerr := c.keys.Refresh(ctx); rerr == nil {
			body, err = c.do(ctx, u)
		}
	}
	if err != nil {
		return err
	}
	if uerr := json.Unmarshal(body, out); uerr != nil {
		return fmt.Errorf("enbw: decode response: %w", uerr)
	}
	return nil
}

func (c *Client) do(ctx context.Context, u string) ([]byte, error) {
	key := c.keys.Key()
	if key == "" {
		return nil, errNoKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("enbw: build request: %w", err)
	}
	req.Header.Set(subKeyHeader, key)
	req.Header.Set("Accept", "application/json")
	// Azure APIM blocks the default Go user-agent and gates responses on a
	// browser-like Origin + Referer. All three are required together — drop
	// any one and the API returns 403 (which looks like, but is not, a
	// throttle). See internal/enbw client tests / AGENTS.md.
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Origin", "https://www.enbw.com")
	req.Header.Set("Referer", "https://www.enbw.com/")
	req.Header.Set("Accept-Language", "de")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enbw: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{status: resp.StatusCode}
	}
	return body, nil
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// httpError is a non-200 response from the API.
type httpError struct{ status int }

func (e *httpError) Error() string { return fmt.Sprintf("enbw: api status %d", e.status) }

var errNoKey = &noKeyError{}

type noKeyError struct{}

func (*noKeyError) Error() string { return "enbw: no subscription key available" }

// isAuthError reports whether err is a 401/403 from the API, which signals an
// expired or throttled key.
func isAuthError(err error) bool {
	he, ok := err.(*httpError)
	return ok && (he.status == http.StatusUnauthorized || he.status == http.StatusForbidden)
}
