package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestAuthFlow walks through what a real user does: register, log in, log out. It sends real
// HTTP requests to the app's router (newMux) running on a throwaway server, with an in-memory
// database, and a cookie jar that keeps the session cookie between requests like a browser.
func TestAuthFlow(t *testing.T) {
	newTestDB(t)

	useTestSessionKey(t)

	site := newTestSite(t)

	serverURL, err := url.Parse(site.server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	jar := site.client.Jar

	// do sends one request and returns the status code and the body. A nil form means no body.
	do := func(method, path string, form url.Values) (int, string) {
		t.Helper()
		status, _, body := site.send(t, method, path, form)
		return status, body
	}

	hasSessionCookie := func() bool {
		for _, c := range jar.Cookies(serverURL) {
			if c.Name == "session" {
				return true
			}
		}
		return false
	}

	registerForm := url.Values{
		"username":  {"alice"},
		"email":     {"alice@example.com"},
		"password":  {"correct-horse"},
		"password2": {"correct-horse"},
	}
	loginForm := func(password string) url.Values {
		return url.Values{"username": {"alice"}, "password": {password}}
	}

	// expectStatus sends one request and fails the step unless it answers with want.
	expectStatus := func(t *testing.T, method, path string, form url.Values, want int) {
		t.Helper()
		if status, body := do(method, path, form); status != want {
			t.Fatalf("%s %s: status = %d, want %d: %s", method, path, status, want, body)
		}
	}

	// expectLoggedIn checks that the jar holds a session cookie and the front page shows #nav-logout.
	expectLoggedIn := func(t *testing.T) {
		t.Helper()
		if !hasSessionCookie() {
			t.Fatal("no session cookie")
		}
		if _, page := do(http.MethodGet, "/", nil); !strings.Contains(page, `id="nav-logout"`) {
			t.Error("front page has no #nav-logout for a logged-in user")
		}
	}

	// expectLoggedOut checks that there is no session cookie and the front page shows #nav-login.
	expectLoggedOut := func(t *testing.T) {
		t.Helper()
		if hasSessionCookie() {
			t.Error("session cookie present for a logged-out user")
		}
		_, page := do(http.MethodGet, "/", nil)
		if strings.Contains(page, `id="nav-logout"`) {
			t.Error("front page shows #nav-logout for a logged-out user")
		}
		if !strings.Contains(page, `id="nav-login"`) {
			t.Error("front page has no #nav-login for a logged-out user")
		}
	}

	// The steps share one user and one cookie jar, so they must run in this order.
	t.Run("register logs the new user in", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/register", registerForm, http.StatusOK)
		expectLoggedIn(t)
	})

	// Log out again, so the login steps below start without a session
	t.Run("logout after register", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		expectLoggedOut(t)
	})

	t.Run("register the same user again is rejected", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/register", registerForm, http.StatusUnprocessableEntity)
		expectLoggedOut(t)
	})

	t.Run("login with wrong password", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/login", loginForm("wrong"), http.StatusUnauthorized)
		expectLoggedOut(t)
	})

	t.Run("login", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/login", loginForm("correct-horse"), http.StatusOK)
		expectLoggedIn(t)
	})

	t.Run("logout", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		expectLoggedOut(t)
	})
}

// TestAboutPage checks that the footer's "About" link leads to a real page. Without a route for
// /about the request fell through to GET / and showed the search page instead.
func TestAboutPage(t *testing.T) {
	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/about", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /about: %v", err)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /about: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	page := string(content)
	if !strings.Contains(page, "Our mission") {
		t.Error("/about does not show the About content")
	}
	if strings.Contains(page, `id="search-input"`) {
		t.Error("/about shows the search page, so the request fell through to GET /")
	}
}

// TestSeedPagesSearchable checks that the legacy pages are loaded, that loading them twice adds
// nothing (the app does it on every startup), and that the search page finds one of them.
func TestSeedPagesSearchable(t *testing.T) {
	conn := newTestDB(t)
	for range 2 {
		if err := seedPages(t.Context(), conn); err != nil {
			t.Fatalf("seed pages: %v", err)
		}
	}

	var count int
	if err := conn.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM pages`).Scan(&count); err != nil {
		t.Fatalf("count pages: %v", err)
	}
	if count != 51 {
		t.Errorf("pages after seeding twice = %d, want 51", count)
	}

	page := newTestSite(t).page(t, "/?q=javascript")
	if !strings.Contains(page, `class="search-result-title"`) || !strings.Contains(page, ">JavaScript</a>") {
		t.Error(`searching "javascript" does not show the JavaScript page`)
	}
}
