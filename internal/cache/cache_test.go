package cache

import (
	"testing"
	"time"
)

func TestTTLMissThenHit(t *testing.T) {
	c := NewTTL[int](time.Minute)
	if _, ok := c.Get("x"); ok {
		t.Fatal("expected miss on empty cache")
	}
	c.Set("x", 42)
	v, ok := c.Get("x")
	if !ok || v != 42 {
		t.Fatalf("expected hit 42, got %v ok=%v", v, ok)
	}
}

func TestTTLExpiry(t *testing.T) {
	c := NewTTL[string](time.Minute)
	base := time.Unix(1_000_000, 0)
	c.now = func() time.Time { return base }
	c.Set("k", "v")
	if _, ok := c.Get("k"); !ok {
		t.Fatal("expected hit before expiry")
	}
	c.now = func() time.Time { return base.Add(2 * time.Minute) }
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after expiry")
	}
}
