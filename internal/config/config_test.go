package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VOLTPILOT_LISTEN_ADDR", "")
	t.Setenv("VOLTPILOT_ENBW_KEY", "")
	t.Setenv("VOLTPILOT_ENBW_MAP_URL", "")
	t.Setenv("VOLTPILOT_KEY_REFRESH_INTERVAL", "")
	cfg := Load()
	if cfg.ListenAddr != defaultListenAddr {
		t.Fatalf("listen addr = %q, want %q", cfg.ListenAddr, defaultListenAddr)
	}
	if cfg.ENBWMapURL != defaultMapURL {
		t.Fatalf("map url = %q, want default", cfg.ENBWMapURL)
	}
	if cfg.KeyRefreshInterval != defaultRefresh {
		t.Fatalf("refresh = %v, want %v", cfg.KeyRefreshInterval, defaultRefresh)
	}
	if cfg.ENBWKeySeed != "" {
		t.Fatalf("expected empty key seed, got %q", cfg.ENBWKeySeed)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("VOLTPILOT_LISTEN_ADDR", ":9999")
	t.Setenv("VOLTPILOT_ENBW_KEY", "seed123")
	t.Setenv("VOLTPILOT_KEY_REFRESH_INTERVAL", "30m")
	cfg := Load()
	if cfg.ListenAddr != ":9999" || cfg.ENBWKeySeed != "seed123" {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
	if cfg.KeyRefreshInterval != 30*time.Minute {
		t.Fatalf("interval = %v, want 30m", cfg.KeyRefreshInterval)
	}
}

func TestLoadIgnoresBadInterval(t *testing.T) {
	t.Setenv("VOLTPILOT_KEY_REFRESH_INTERVAL", "not-a-duration")
	if got := Load().KeyRefreshInterval; got != defaultRefresh {
		t.Fatalf("bad interval should fall back to default, got %v", got)
	}
}
