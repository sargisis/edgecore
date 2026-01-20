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

	// Request duration histogram (seconds).
	fmt.Fprintf(w, "# HELP edgecore_request_duration_seconds Request duration in seconds\n")
	fmt.Fprintf(w, "# TYPE edgecore_request_duration_seconds histogram\n")

	var count uint64
	for i, upper := range latencyBuckets {
		bucketCount := atomic.LoadUint64(&latencyCounts[i])
		count += bucketCount
		fmt.Fprintf(w, "edgecore_request_duration_seconds_bucket{le=\"%.3f\"} %d\n", upper, count)
	}

	// +Inf bucket.
	infCount := atomic.LoadUint64(&latencyCounts[len(latencyCounts)-1])
	count += infCount
	fmt.Fprintf(w, "edgecore_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", count)

	// Sum and count.
	sumSeconds := float64(atomic.LoadUint64(&latencySumMicros)) / 1e6
	fmt.Fprintf(w, "edgecore_request_duration_seconds_sum %f\n", sumSeconds)
	fmt.Fprintf(w, "edgecore_request_duration_seconds_count %d\n", count)
}
