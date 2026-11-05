package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
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

	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)

	serverURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	// do sends one request and returns the status code and the body. A nil form means no body.
	do := func(method, path string, form url.Values) (int, string) {
		t.Helper()
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, body)
		if err != nil {
			t.Fatalf("build %s %s: %v", method, path, err)
		}
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := client.Do(req)
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
		return resp.StatusCode, string(content)
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
