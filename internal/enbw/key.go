package enbw

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
)

// keyPattern matches the 32-hex subscription key embedded in the EnBW map
// page's inline config: apimSubscriptionKey: "d495...".
var keyPattern = regexp.MustCompile(`apimSubscriptionKey:\s*"([0-9a-f]{32})"`)

// KeyManager holds the current EnBW subscription key and refreshes it by
// scraping the public map page. The key is shared and rotates, so we never
// hardcode it in tracked source — it comes from a seed env var and/or the
// scrape.
type KeyManager struct {
	mapURL string
	http   *http.Client

	mu  sync.RWMutex
	key string
}

// NewKeyManager builds a KeyManager seeded with an optional initial key.
func NewKeyManager(mapURL string, seed string, hc *http.Client) *KeyManager {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &KeyManager{mapURL: mapURL, http: hc, key: seed}
}

// Key returns the current subscription key, which may be empty before the
// first successful seed/refresh.
func (m *KeyManager) Key() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.key
}

func (m *KeyManager) setKey(k string) {
	m.mu.Lock()
	m.key = k
	m.mu.Unlock()
}

// Refresh scrapes the map page and updates the key on success. It returns an
// error without mutating the current key when the page can't be fetched or no
// key is found, so a transient failure never wipes a working key.
func (m *KeyManager) Refresh(ctx context.Context) error {
	key, err := m.scrape(ctx)
	if err != nil {
		return err
	}
	prev := m.Key()
	m.setKey(key)
	if key != prev {
		slog.Info("enbw subscription key refreshed", "changed", prev != "")
	}
	return nil
}

func (m *KeyManager) scrape(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.mapURL, nil)
	if err != nil {
		return "", fmt.Errorf("enbw key: build request: %w", err)
	}
	// A browser-like UA avoids bot interstitials on the marketing page.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; voltpilot/1.0)")
	resp, err := m.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("enbw key: fetch map page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("enbw key: map page status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("enbw key: read map page: %w", err)
	}
	return ExtractKey(string(body))
}

// ExtractKey pulls the subscription key out of map-page HTML/JS. Exported for
// testing and reuse.
func ExtractKey(html string) (string, error) {
	if mch := keyPattern.FindStringSubmatch(html); len(mch) == 2 {
		return mch[1], nil
	}
	return "", fmt.Errorf("enbw key: no apimSubscriptionKey found in page")
}
