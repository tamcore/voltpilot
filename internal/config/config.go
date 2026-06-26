// Package config loads server configuration from the environment.
package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for the voltpilot server.
type Config struct {
	// ListenAddr is the HTTP listen address, e.g. ":8080".
	ListenAddr string
	// ENBWKeySeed is an optional subscription key used until (and as a
	// fallback if) the dynamic scrape from the EnBW map page succeeds.
	ENBWKeySeed string
	// ENBWMapURL is the page scraped for the rotating subscription key.
	ENBWMapURL string
	// KeyRefreshInterval is how often the key is re-scraped in the background.
	KeyRefreshInterval time.Duration
}

const (
	defaultListenAddr = ":8080"
	defaultMapURL     = "https://www.enbw.com/elektromobilitaet/produkte/mobilityplus-app/ladestation-finden/map"
	defaultRefresh    = 6 * time.Hour
)

// Load reads configuration from the environment, applying defaults. It never
// fails: a missing key seed is fine because the background scraper supplies it.
func Load() Config {
	cfg := Config{
		ListenAddr:         envOr("VOLTPILOT_LISTEN_ADDR", defaultListenAddr),
		ENBWKeySeed:        os.Getenv("VOLTPILOT_ENBW_KEY"),
		ENBWMapURL:         envOr("VOLTPILOT_ENBW_MAP_URL", defaultMapURL),
		KeyRefreshInterval: defaultRefresh,
	}
	if v := os.Getenv("VOLTPILOT_KEY_REFRESH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.KeyRefreshInterval = d
		}
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
