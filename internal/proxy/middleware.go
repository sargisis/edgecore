package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// Latency histogram buckets (in seconds) and counters for request duration.
var (
	latencyBuckets = []float64{
		0.005, 0.01, 0.025, 0.05, 0.1,
		0.25, 0.5, 1.0, 2.5, 5.0,
	}
	// latencyCounts holds counts for each bucket plus one extra for +Inf.
	latencyCounts = make([]uint64, len(latencyBuckets)+1)
	// latencySumMicros accumulates total request duration in microseconds.
	latencySumMicros uint64
)

// recordLatency updates the latency histogram for a given duration.
func recordLatency(d time.Duration) {
	seconds := d.Seconds()

	// Sum in microseconds to avoid float atomics.
	micros := uint64(d / time.Microsecond)
	atomic.AddUint64(&latencySumMicros, micros)

	// Find appropriate bucket.
	idx := len(latencyBuckets) // default is +Inf bucket
	for i, bound := range latencyBuckets {
		if seconds <= bound {
			idx = i
			break
		}
	}
	atomic.AddUint64(&latencyCounts[idx], 1)
}

// generateRequestID creates a short random ID for request tracking.
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Logger is a middleware to log requests with structured logging and request IDs.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		atomic.AddUint64(&GlobalMetrics.TotalRequests, 1)

		// Generate request ID and add it to response header
		requestID := generateRequestID()
		if r.Header.Get("X-Request-ID") == "" {
			r.Header.Set("X-Request-ID", requestID)
		} else {
			requestID = r.Header.Get("X-Request-ID")
		}
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		dur := time.Since(start)
		recordLatency(dur)

		// Extract client IP
		clientIP := clientIP(r)

		// Extract backend URL if available
		backendURL := r.Header.Get("X-Backend-URL")

		// Log structured entry
		entry := LogEntry{
			Level:     "info",
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    rw.statusCode,
			Duration:  dur.Seconds(),
			ClientIP:  clientIP,
			RequestID: requestID,
		}
		if backendURL != "" {
			entry.Backend = backendURL
		}
		logEntry(entry)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
