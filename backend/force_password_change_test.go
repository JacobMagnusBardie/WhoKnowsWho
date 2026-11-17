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
)

// TestMigrateSchemaAddsMustChangePassword checks that a database created before the
// must_change_password column existed gets the column, without losing its users, and that
// running the migration again changes nothing.
func TestMigrateSchemaAddsMustChangePassword(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	// The users table as it was before the column existed.
	if _, err := conn.ExecContext(t.Context(), `CREATE TABLE users (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email    TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create old users table: %v", err)
	}
	if _, err := conn.ExecContext(t.Context(), `INSERT INTO users (username, email, password) VALUES ('old', 'old@example.com', 'hash')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	for range 2 { // the second run must be a no-op
		if err := migrateSchema(t.Context(), conn); err != nil {
			t.Fatalf("migrateSchema: %v", err)
		}
	}

	var flag int
	if err := conn.QueryRowContext(t.Context(), `SELECT must_change_password FROM users WHERE username = 'old'`).Scan(&flag); err != nil {
		t.Fatalf("read flag of the existing user: %v", err)
	}
	if flag != 0 {
		t.Errorf("must_change_password = %d for an existing user, want 0", flag)
	}
}

// TestForcedPasswordChange walks through the data breach scenario: a user's credentials leaked,
// the account is flagged, and the user can do nothing but change the password. It sends real HTTP
// requests to the router on a throwaway server with an in-memory database, and a cookie jar that
// keeps the session cookie between requests like a browser. The steps share one user and one
// jar, so they must run in this order.
func TestForcedPasswordChange(t *testing.T) {
	conn := newTestDB(t)
	useTestSessionKey(t)

	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		// Don't follow redirects, so the steps can check the redirect itself.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	// send sends one request and returns the status code, the Location header and the body.
	// A nil form means no body.
	send := func(t *testing.T, method, path string, form url.Values) (status int, location, body string) {
		t.Helper()
		var reqBody io.Reader
		if form != nil {
			reqBody = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, reqBody)
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
		return resp.StatusCode, resp.Header.Get("Location"), string(content)
	}

	// expectStatus sends one request and fails the step unless it answers with want.
	expectStatus := func(t *testing.T, method, path string, form url.Values, want int) {
		t.Helper()
		if status, _, body := send(t, method, path, form); status != want {
			t.Fatalf("%s %s: status = %d, want %d: %s", method, path, status, want, body)
		}
	}

	// expectRedirectToChangePassword checks that a page sends the user on to /change-password.
	expectRedirectToChangePassword := func(t *testing.T, path string) {
		t.Helper()
		status, location, body := send(t, http.MethodGet, path, nil)
		if status != http.StatusSeeOther || location != "/change-password" {
			t.Fatalf("GET %s: status = %d, Location = %q, want 303 to /change-password: %s", path, status, location, body)
		}
	}

	changeForm := func(oldPassword, newPassword, newPassword2 string) url.Values {
		return url.Values{
			"old_password":  {oldPassword},
			"new_password":  {newPassword},
			"new_password2": {newPassword2},
		}
	}
	loginForm := func(password string) url.Values {
		return url.Values{"username": {"bob"}, "password": {password}}
	}

	const oldPassword = "leaked-password"
	const newPassword = "brand-new-password"

	t.Run("register a user and flag the account", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/register", url.Values{
			"username": {"bob"},
			"email":    {"bob@example.com"},
			"password": {oldPassword},
		}, http.StatusOK)
		// A user who is not flagged can use the site as usual.
		expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)

		// This is what scripts/flag-compromised-users.sh does for the leaked users.
		if _, err := conn.ExecContext(t.Context(), `UPDATE users SET must_change_password = 1 WHERE username = ?`, "bob"); err != nil {
			t.Fatalf("flag user: %v", err)
		}
	})

	t.Run("a flagged user is sent to the change-password page", func(t *testing.T) {
		expectRedirectToChangePassword(t, "/")
		expectRedirectToChangePassword(t, "/register")
	})

	t.Run("a flagged user is refused the API", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/search?q=test", nil, http.StatusForbidden)
	})

	t.Run("the change-password page explains why and still loads its styling", func(t *testing.T) {
		status, _, page := send(t, http.MethodGet, "/change-password", nil)
		if status != http.StatusOK {
			t.Fatalf("GET /change-password: status = %d, want 200", status)
		}
		if !strings.Contains(page, "data breach") {
			t.Error("change-password page does not tell the user about the breach")
		}
		if !strings.Contains(page, `action="/api/change-password"`) {
			t.Error("change-password page has no form for /api/change-password")
		}
		expectStatus(t, http.MethodGet, "/static/style.css", nil, http.StatusOK)
	})

	t.Run("a flagged user who logs in again is still sent to the change-password page", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		expectStatus(t, http.MethodPost, "/api/login", loginForm(oldPassword), http.StatusOK)
		expectRedirectToChangePassword(t, "/")
	})

	t.Run("a bad password change is rejected and the user stays flagged", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm("wrong", newPassword, newPassword), http.StatusUnprocessableEntity)
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm(oldPassword, newPassword, "other"), http.StatusUnprocessableEntity)
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm(oldPassword, "", ""), http.StatusUnprocessableEntity)
		// The old password may not be reused
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm(oldPassword, oldPassword, oldPassword), http.StatusUnprocessableEntity)
		expectRedirectToChangePassword(t, "/")
	})

	t.Run("changing the password unlocks the site", func(t *testing.T) {
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm(oldPassword, newPassword, newPassword), http.StatusOK)
		expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)
		expectStatus(t, http.MethodGet, "/api/search?q=test", nil, http.StatusOK)

		var flag int
		if err := conn.QueryRowContext(t.Context(), `SELECT must_change_password FROM users WHERE username = ?`, "bob").Scan(&flag); err != nil {
			t.Fatalf("read flag: %v", err)
		}
		if flag != 0 {
			t.Errorf("must_change_password = %d after the change, want 0", flag)
		}
	})

	t.Run("only the new password works afterwards", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		expectStatus(t, http.MethodPost, "/api/login", loginForm(oldPassword), http.StatusUnauthorized)
		expectStatus(t, http.MethodPost, "/api/login", loginForm(newPassword), http.StatusOK)
		expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)
	})

	t.Run("changing the password needs a login", func(t *testing.T) {
		expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		expectStatus(t, http.MethodPost, "/api/change-password", changeForm(newPassword, "another-password", "another-password"), http.StatusUnauthorized)
	})
}
