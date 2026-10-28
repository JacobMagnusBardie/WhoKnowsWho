package main

import (
	"database/sql"
	"testing"
)

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
