package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPFromXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	req.RemoteAddr = "192.0.2.1:1234"

	ip := clientIP(req)
	if ip != "203.0.113.1" {
		t.Fatalf("expected client IP from X-Forwarded-For to be 203.0.113.1, got %q", ip)
	}
}

func TestClientIPFromXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Real-IP", "198.51.100.2")
	req.RemoteAddr = "192.0.2.1:1234"

	ip := clientIP(req)
	if ip != "198.51.100.2" {
		t.Fatalf("expected client IP from X-Real-IP to be 198.51.100.2, got %q", ip)
	}
}

func TestClientIPFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	ip := clientIP(req)
	if ip != "192.0.2.1" {
		t.Fatalf("expected client IP from RemoteAddr to be 192.0.2.1, got %q", ip)
	}
}

func TestIPRateLimitMiddlewareBlocksPerIP(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1) // 1 request per second per IP

	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})

	handler := IPRateLimitMiddleware(limiter, next)

	// First request from the same IP should pass.
	req1 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req1.RemoteAddr = "192.0.2.1:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got status %d", rr1.Code)
	}

	// Second immediate request from the same IP should be rate limited.
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req2.RemoteAddr = "192.0.2.1:5678"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got status %d", rr2.Code)
	}

	// A different IP should still be allowed.
	req3 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req3.RemoteAddr = "198.51.100.5:9999"
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected request from different IP to succeed, got status %d", rr3.Code)
	}

	if calls != 2 {
		t.Fatalf("expected handler to be called twice, got %d", calls)
	}
}
