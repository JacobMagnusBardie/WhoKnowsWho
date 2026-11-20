package main

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/securecookie"
)

// useTestSessionKey gives the handlers a random session signing key for one test. main()
// normally sets sessionKey; register and login need it to set the session cookie.
func useTestSessionKey(t *testing.T) {
	t.Helper()
	previousKey := sessionKey
	sessionKey = securecookie.New(securecookie.GenerateRandomKey(32), nil)
	t.Cleanup(func() { sessionKey = previousKey })
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	conn.SetMaxOpenConns(1)
	previousDB := db
	db = conn
	t.Cleanup(func() {
		db = previousDB
		if err := conn.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if _, err := conn.ExecContext(t.Context(), schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	return conn
}

// testSite is the app running on a throwaway server, with a cookie jar that keeps the session
// cookie between requests like a browser. It does not follow redirects, so tests can check them.
type testSite struct {
	server *httptest.Server
	client *http.Client
}

func newTestSite(t *testing.T) *testSite {
	t.Helper()
	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	return &testSite{
		server: srv,
		client: &http.Client{
			Jar:           jar,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// send sends one request and returns the status code, the Location header and the body.
// A nil form means no body.
func (s *testSite) send(t *testing.T, method, path string, form url.Values) (status int, location, body string) {
	t.Helper()
	var reqBody io.Reader
	if form != nil {
		reqBody = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(t.Context(), method, s.server.URL+path, reqBody)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response of %s %s: %v", method, path, err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close response of %s %s: %v", method, path, err)
	}
	return resp.StatusCode, resp.Header.Get("Location"), string(content)
}

// expectStatus sends one request and fails the test unless it answers with want.
func (s *testSite) expectStatus(t *testing.T, method, path string, form url.Values, want int) {
	t.Helper()
	if status, _, body := s.send(t, method, path, form); status != want {
		t.Fatalf("%s %s: status = %d, want %d: %s", method, path, status, want, body)
	}
}

// expectRedirect checks that a GET answers 303 See Other with the given Location.
func (s *testSite) expectRedirect(t *testing.T, path, wantLocation string) {
	t.Helper()
	status, location, body := s.send(t, http.MethodGet, path, nil)
	if status != http.StatusSeeOther || location != wantLocation {
		t.Fatalf("GET %s: status = %d, Location = %q, want 303 to %s: %s", path, status, location, wantLocation, body)
	}
}

// page fetches a page that must answer 200 and returns its HTML.
func (s *testSite) page(t *testing.T, path string) string {
	t.Helper()
	status, _, body := s.send(t, http.MethodGet, path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s: status = %d, want 200: %s", path, status, body)
	}
	return body
}
