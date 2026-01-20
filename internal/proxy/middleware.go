package proxy

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds the data for the load balancer performance
type Metrics struct {
	TotalRequests uint64
	RateLimited   uint64
}

var GlobalMetrics Metrics

// Logger is a middleware to log requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		atomic.AddUint64(&GlobalMetrics.TotalRequests, 1)

		next.ServeHTTP(w, r)

		log.Printf("[%s] %s %s took %v", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

// RateLimitMiddleware applies a global rate limit to all requests.
// Kept for backwards compatibility; new code should prefer IPRateLimitMiddleware.
func RateLimitMiddleware(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			atomic.AddUint64(&GlobalMetrics.RateLimited, 1)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IPRateLimiter manages a set of RateLimiters keyed by client IP.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*RateLimiter
	rate     float64
	capacity float64
}

// NewIPRateLimiter creates a new IP-based rate limiter.
func NewIPRateLimiter(rate, capacity float64) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rate:     rate,
		capacity: capacity,
	}
}

// getLimiter returns the RateLimiter for the given IP, creating one if needed.
func (l *IPRateLimiter) getLimiter(ip string) *RateLimiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, ok := l.limiters[ip]
	if !ok {
		limiter = NewRateLimiter(l.rate, l.capacity)
		l.limiters[ip] = limiter
	}
	return limiter
}

// clientIP attempts to determine the real client IP, taking into account
// common proxy headers. This is a best-effort implementation and assumes
// that the deployment sits behind trusted proxies that set these headers.
func clientIP(r *http.Request) string {
	// X-Forwarded-For may contain a comma-separated list of IPs.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// X-Real-IP is often set by reverse proxies.
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fallback to RemoteAddr (strip port if present).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// IPRateLimitMiddleware applies rate limiting per client IP.
func IPRateLimitMiddleware(limiter *IPRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !limiter.getLimiter(ip).Allow() {
			atomic.AddUint64(&GlobalMetrics.RateLimited, 1)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
