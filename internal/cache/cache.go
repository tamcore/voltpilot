// Package cache provides a tiny in-process TTL cache. The EnBW data is
// ephemeral and cheap to refetch, so we only need short-lived memoization to
// smooth out polling bursts and stay under the API's throttle — no Redis.
package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value   V
	expires time.Time
}

// TTL is a concurrency-safe map with per-entry expiry.
type TTL[V any] struct {
	ttl  time.Duration
	now  func() time.Time
	mu   sync.Mutex
	data map[string]entry[V]
}

// NewTTL builds a TTL cache with the given lifetime.
func NewTTL[V any](ttl time.Duration) *TTL[V] {
	return &TTL[V]{ttl: ttl, now: time.Now, data: make(map[string]entry[V])}
}

// Get returns the cached value for key when present and unexpired.
func (c *TTL[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok || c.now().After(e.expires) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores value under key with the configured TTL.
func (c *TTL[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = entry[V]{value: value, expires: c.now().Add(c.ttl)}
}
