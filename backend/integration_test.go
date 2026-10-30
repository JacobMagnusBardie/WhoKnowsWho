package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/securecookie"
)

// TestAuthFlow walks through what a real user does: register, log in, log out. It sends real
// HTTP requests to the app's router (newMux) running on a throwaway server, with an in-memory
// database, and a cookie jar that keeps the session cookie between requests like a browser.
func TestAuthFlow(t *testing.T) {
	newTestDB(t)

	// The handlers sign the session cookie with the global sessionKey, which main() normally sets.
	previousKey := sessionKey
	sessionKey = securecookie.New(securecookie.GenerateRandomKey(32), nil)
	t.Cleanup(func() { sessionKey = previousKey })

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

	// The steps share one user and one cookie jar, so they must run in this order.
	t.Run("register", func(t *testing.T) {
		if status, body := do(http.MethodPost, "/api/register", registerForm); status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, body)
		}
	})

	t.Run("register the same user again is rejected", func(t *testing.T) {
		if status, body := do(http.MethodPost, "/api/register", registerForm); status != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422: %s", status, body)
		}
	})

	t.Run("login with wrong password", func(t *testing.T) {
		if status, body := do(http.MethodPost, "/api/login", loginForm("wrong")); status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401: %s", status, body)
		}
		if hasSessionCookie() {
			t.Error("session cookie was set after a failed login")
		}
	})

	t.Run("login", func(t *testing.T) {
		if status, body := do(http.MethodPost, "/api/login", loginForm("correct-horse")); status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, body)
		}
		if !hasSessionCookie() {
			t.Fatal("no session cookie after login")
		}
		_, page := do(http.MethodGet, "/", nil)
		if !strings.Contains(page, `id="nav-logout"`) {
			t.Error("front page has no #nav-logout for a logged-in user")
		}
	})

	t.Run("logout", func(t *testing.T) {
		if status, body := do(http.MethodGet, "/api/logout", nil); status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, body)
		}
		if hasSessionCookie() {
			t.Error("session cookie still present after logout")
		}
		_, page := do(http.MethodGet, "/", nil)
		if strings.Contains(page, `id="nav-logout"`) {
			t.Error("front page still shows #nav-logout after logout")
		}
		if !strings.Contains(page, `id="nav-login"`) {
			t.Error("front page has no #nav-login after logout")
		}
	})
}
