package proxy

import (
	"log"
	"net/http"
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

// RateLimitMiddleware applies rate limiting to requests
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
