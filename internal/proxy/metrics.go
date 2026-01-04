package proxy

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// PrometheusMetrics exposes metrics in Prometheus format
func PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP edgecore_requests_total Total number of requests\n")
	fmt.Fprintf(w, "# TYPE edgecore_requests_total counter\n")
	fmt.Fprintf(w, "edgecore_requests_total %d\n", atomic.LoadUint64(&GlobalMetrics.TotalRequests))

	fmt.Fprintf(w, "# HELP edgecore_rate_limited_total Total number of rate limited requests\n")
	fmt.Fprintf(w, "# TYPE edgecore_rate_limited_total counter\n")
	fmt.Fprintf(w, "edgecore_rate_limited_total %d\n", atomic.LoadUint64(&GlobalMetrics.RateLimited))
}
