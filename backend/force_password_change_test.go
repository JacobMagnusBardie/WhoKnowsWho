package main

import (
	"database/sql"
	"net/http"
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
// the account is flagged, and the user can do nothing but change the password. The steps share
// one user and one cookie jar, so they must run in this order.
func TestForcedPasswordChange(t *testing.T) {
	conn := newTestDB(t)
	useTestSessionKey(t)
	site := newTestSite(t)

	const oldPassword = "leaked-password"
	const newPassword = "brand-new-password"

	t.Run("register a user and flag the account", func(t *testing.T) {
		site.expectStatus(t, http.MethodPost, "/api/register", url.Values{
			"username": {"bob"},
			"email":    {"bob@example.com"},
			"password": {oldPassword},
		}, http.StatusOK)
		// A user who is not flagged can use the site as usual.
		site.expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)

		// This is what scripts/flag-compromised-users.sh does for the leaked users.
		setMustChangePassword(t, conn, "bob", 1)
	})

	t.Run("a flagged user is sent to the change-password page", func(t *testing.T) {
		site.expectRedirect(t, "/", "/change-password")
		site.expectRedirect(t, "/register", "/change-password")
	})

	t.Run("a flagged user is refused the API", func(t *testing.T) {
		site.expectStatus(t, http.MethodGet, "/api/search?q=test", nil, http.StatusForbidden)
	})

	t.Run("the change-password page explains why and still loads its styling", func(t *testing.T) {
		assertChangePasswordPage(t, site.page(t, "/change-password"))
		site.expectStatus(t, http.MethodGet, "/static/style.css", nil, http.StatusOK)
	})

	t.Run("a flagged user who logs in again is still sent to the change-password page", func(t *testing.T) {
		site.expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		site.expectStatus(t, http.MethodPost, "/api/login", bobLogin(oldPassword), http.StatusOK)
		site.expectRedirect(t, "/", "/change-password")
	})

	t.Run("a bad password change is rejected and the user stays flagged", func(t *testing.T) {
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange("wrong", newPassword, newPassword), http.StatusUnprocessableEntity)
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange(oldPassword, newPassword, "other"), http.StatusUnprocessableEntity)
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange(oldPassword, "", ""), http.StatusUnprocessableEntity)
		// The old password may not be reused
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange(oldPassword, oldPassword, oldPassword), http.StatusUnprocessableEntity)
		site.expectRedirect(t, "/", "/change-password")
	})

	t.Run("changing the password unlocks the site", func(t *testing.T) {
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange(oldPassword, newPassword, newPassword), http.StatusOK)
		site.expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)
		site.expectStatus(t, http.MethodGet, "/api/search?q=test", nil, http.StatusOK)

		if flag := mustChangePassword(t, conn, "bob"); flag != 0 {
			t.Errorf("must_change_password = %d after the change, want 0", flag)
		}
	})

	t.Run("only the new password works afterwards", func(t *testing.T) {
		site.expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		site.expectStatus(t, http.MethodPost, "/api/login", bobLogin(oldPassword), http.StatusUnauthorized)
		site.expectStatus(t, http.MethodPost, "/api/login", bobLogin(newPassword), http.StatusOK)
		site.expectStatus(t, http.MethodGet, "/", nil, http.StatusOK)
	})

	t.Run("changing the password needs a login", func(t *testing.T) {
		site.expectStatus(t, http.MethodGet, "/api/logout", nil, http.StatusOK)
		site.expectStatus(t, http.MethodPost, "/api/change-password", passwordChange(newPassword, "another-password", "another-password"), http.StatusUnauthorized)
	})
}

// passwordChange builds the form for /api/change-password.
func passwordChange(oldPassword, newPassword, newPassword2 string) url.Values {
	return url.Values{
		"old_password":  {oldPassword},
		"new_password":  {newPassword},
		"new_password2": {newPassword2},
	}
}

// bobLogin builds the login form for the user of TestForcedPasswordChange.
func bobLogin(password string) url.Values {
	return url.Values{"username": {"bob"}, "password": {password}}
}

// setMustChangePassword sets the flag of one user directly in the database.
func setMustChangePassword(t *testing.T, conn *sql.DB, username string, flag int) {
	t.Helper()
	if _, err := conn.ExecContext(t.Context(), `UPDATE users SET must_change_password = ? WHERE username = ?`, flag, username); err != nil {
		t.Fatalf("set must_change_password: %v", err)
	}
}

// mustChangePassword reads the flag of one user from the database.
func mustChangePassword(t *testing.T, conn *sql.DB, username string) int {
	t.Helper()
	var flag int
	if err := conn.QueryRowContext(t.Context(), `SELECT must_change_password FROM users WHERE username = ?`, username).Scan(&flag); err != nil {
		t.Fatalf("read must_change_password: %v", err)
	}
	return flag
}

// assertChangePasswordPage checks that the page tells the user about the breach and has the form.
func assertChangePasswordPage(t *testing.T, page string) {
	t.Helper()
	if !strings.Contains(page, "data breach") {
		t.Error("change-password page does not tell the user about the breach")
	}
	if !strings.Contains(page, `action="/api/change-password"`) {
		t.Error("change-password page has no form for /api/change-password")
	}
}
