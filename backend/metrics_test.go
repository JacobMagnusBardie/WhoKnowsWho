package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHealth(t *testing.T) {
	newTestDB(t)
	page := newTestSite(t).page(t, "/health")
	if !strings.Contains(page, `"status":"ok"`) {
		t.Errorf("GET /health body = %q, want status ok", page)
	}
}

// TestMetricsCountRequestsAndSearches checks that a search is counted both as a request on its
// route and as an empty or successful search. Counters are global, so it compares before/after.
func TestMetricsCountRequestsAndSearches(t *testing.T) {
	conn := newTestDB(t)
	if _, err := conn.ExecContext(t.Context(),
		`INSERT INTO pages (title, url, content, language) VALUES ('Go', 'https://go.dev', 'Go is a language', 'en')`); err != nil {
		t.Fatalf("insert page: %v", err)
	}
	site := newTestSite(t)

	requestsBefore := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET /api/search", "GET", "200"))
	hitsBefore := testutil.ToFloat64(searchesTotal.WithLabelValues("hits"))
	emptyBefore := testutil.ToFloat64(searchesTotal.WithLabelValues("empty"))

	site.expectStatus(t, http.MethodGet, "/api/search?q=language", nil, http.StatusOK)
	site.expectStatus(t, http.MethodGet, "/api/search?q=nothingmatchesthis", nil, http.StatusOK)

	if got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET /api/search", "GET", "200")) - requestsBefore; got != 2 {
		t.Errorf("GET /api/search requests counted = %v, want 2", got)
	}
	if got := testutil.ToFloat64(searchesTotal.WithLabelValues("hits")) - hitsBefore; got != 1 {
		t.Errorf("searches with hits counted = %v, want 1", got)
	}
	if got := testutil.ToFloat64(searchesTotal.WithLabelValues("empty")) - emptyBefore; got != 1 {
		t.Errorf("empty searches counted = %v, want 1", got)
	}
}

// TestMetricsEndpoint checks that /metrics answers on the metrics server and lists our metrics.
func TestMetricsEndpoint(t *testing.T) {
	searchesTotal.WithLabelValues("hits") // make sure the metric has at least one series
	srv := httptest.NewServer(newMetricsServer().Handler)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/metrics", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck // test cleanup
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics: status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /metrics: %v", err)
	}
	if !strings.Contains(string(body), "whoknows_searches_total") {
		t.Error("/metrics does not list whoknows_searches_total")
	}
}
