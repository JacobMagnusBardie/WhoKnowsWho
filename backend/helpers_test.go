package main

import (
	"database/sql"
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
