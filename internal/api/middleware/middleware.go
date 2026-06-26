// Package middleware contains HTTP middlewares used by internal/api.
package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// AccessLog logs one slog record per request. Status >= 500 logs at WARN.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelWarn
		}
		slog.LogAttrs(r.Context(), level, "http",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.Int("status", status),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Int64("dur_ms", time.Since(start).Milliseconds()),
			slog.String("remote", r.RemoteAddr),
			slog.String("request_id", chimw.GetReqID(r.Context())),
		)
	})
}

// SecurityHeaders sets a defensive default header set on every response. The
// CSP allows the OSM tile servers (Leaflet map) and @fontsource data: URIs.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(self)")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data: https://*.tile.openstreetmap.org https://tile.openstreetmap.org; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; manifest-src 'self'")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// RealIP rewrites RemoteAddr from proxy headers, but only when the connection
// originates from a private/loopback peer, so a direct public client cannot
// spoof its own address.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := realClientIP(r); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func realClientIP(r *http.Request) string {
	connIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if connIP == "" {
		connIP = r.RemoteAddr
	}
	if !isPrivateOrLoopback(connIP) {
		return ""
	}
	if v := r.Header.Get("X-Real-IP"); v != "" && net.ParseIP(strings.TrimSpace(v)) != nil {
		return strings.TrimSpace(v)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			if candidate := strings.TrimSpace(parts[i]); net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	return ""
}

var privateRanges = []net.IPNet{
	mustCIDR("10.0.0.0/8"), mustCIDR("172.16.0.0/12"), mustCIDR("192.168.0.0/16"),
	mustCIDR("fc00::/7"), mustCIDR("127.0.0.0/8"), mustCIDR("::1/128"),
}

func isPrivateOrLoopback(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func mustCIDR(s string) net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return *n
}
