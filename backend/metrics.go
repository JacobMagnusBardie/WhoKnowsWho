package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// The metrics below are counters and histograms kept in the app's memory. Prometheus fetches
// their current values from /metrics on metricsAddr and stores them over time.
// Docs/monitoring/metrics.md explains which question each metric answers.
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "whoknows_http_requests_total",
		Help: "HTTP requests handled, by route, method and status code.",
	}, []string{"route", "method", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "whoknows_http_request_duration_seconds",
		Help:    "Time spent handling HTTP requests, by route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})

	searchesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "whoknows_searches_total",
		Help: `Searches, by whether they found anything ("hits") or not ("empty").`,
	}, []string{"result"})

	registrationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "whoknows_registrations_total",
		Help: "Users registered.",
	})

	loginsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "whoknows_logins_total",
		Help: `Login attempts, by result ("ok" or "failed").`,
	}, []string{"result"})
)

// countSearch records one search and whether it found anything.
func countSearch(results []SearchResult) {
	if len(results) == 0 {
		searchesTotal.WithLabelValues("empty").Inc()
		return
	}
	searchesTotal.WithLabelValues("hits").Inc()
}

// statusRecorder remembers the status code a handler writes, so the middleware can label by it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// instrument counts and times every request. The route label is the matched ServeMux pattern
// (e.g. "GET /api/search"), never the raw URL, so random paths can't create endless label values.
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// ServeMux fills in r.Pattern when it matches a route. It stays empty when a middleware
		// answered before the mux, e.g. requirePasswordChange blocking a flagged user.
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		httpRequestsTotal.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
}

// metricsAddr is where /metrics is served. It is a separate port from the site (8080) because
// 8080 is published to the internet. compose does not publish this port, so only containers on
// the same Docker network (Prometheus) can reach it.
func metricsAddr() string {
	if addr := os.Getenv("METRICS_ADDR"); addr != "" {
		return addr
	}
	return ":9091"
}

// newMetricsServer serves /metrics on metricsAddr.
func newMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	return &http.Server{
		Addr:              metricsAddr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
