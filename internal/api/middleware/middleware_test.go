package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing CSP")
	}
	// No TLS on the request → no HSTS.
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should be absent on plain HTTP")
	}
}

func TestAccessLogPassesThrough(t *testing.T) {
	h := AccessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x?y=1", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rec.Code)
	}
}

func TestRealIPRewritesForPrivatePeer(t *testing.T) {
	var seen string
	h := RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.RemoteAddr
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234" // private proxy
	req.Header.Set("X-Real-IP", "203.0.113.7")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "203.0.113.7" {
		t.Fatalf("expected rewrite to client IP, got %q", seen)
	}
}

func TestRealIPIgnoresPublicPeer(t *testing.T) {
	var seen string
	h := RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.RemoteAddr
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:5555" // public peer
	req.Header.Set("X-Real-IP", "10.0.0.1")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "203.0.113.9:5555" {
		t.Fatalf("public peer must not be overridden, got %q", seen)
	}
}
