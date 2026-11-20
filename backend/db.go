package main

import (
	"context"
	"database/sql"
	_ "embed" // for the //go:embed seed file
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite" // pure-Go SQLite driver.
)

// checks for provided path, if any, return it. if not, return local path "whoknows.db"
func dbPath() string {
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	return "whoknows.db"
}

// db is the shared database handle, initialized by initDB() at startup.
var db *sql.DB

// schema defines the tables the app expects to exist. It is applied on every
// startup with CREATE TABLE IF NOT EXISTS, so it is safe to run repeatedly and
// requires no separate migration step for this stage of the project.
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	email    TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	-- 1 when the user must choose a new password before using the site (e.g. after a data breach)
	must_change_password INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pages (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	title    TEXT NOT NULL UNIQUE,
	url      TEXT NOT NULL UNIQUE,
	content  TEXT NOT NULL,
	language TEXT NOT NULL DEFAULT 'en'
);
`

// migrateSchema brings a database that already exists up to date with schema. CREATE TABLE IF
// NOT EXISTS never changes a table that exists, so every column added to schema later has to be
// added here too. Each step checks first, so it is safe to run on every startup. This is a
// stopgap until the project uses a real migration tool.
func migrateSchema(ctx context.Context, conn *sql.DB) error {
	var found int
	err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'must_change_password'`).Scan(&found)
	if err != nil {
		return err
	}
	if found > 0 {
		return nil
	}
	_, err = conn.ExecContext(ctx, `ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0`)
	return err
}

// initDB opens (creating if necessary) the SQLite database file at dbPath and
// ensures the expected tables exist. It is called once from main() before the
// server starts handling requests.

// Go doesn't use try/catch, instead it uses if err != nil to check for errors.
// := initialize both conn and err, so they are available in the if block.
// If err is not nil, it logs a fatal error and exits the program.
// This ensures that the database connection is established successfully before proceeding.
func initDB() *sql.DB {
	path := dbPath()
	conn, err := sql.Open("sqlite", path)
	if err != nil { // function returns nil for its error if it succeeds.
		log.Fatalf("failed to open database %q: %v", path, err)
	}

	if err := conn.Ping(); err != nil {
		log.Fatalf("failed to connect to database %q: %v", path, err)
	}

	if _, err := conn.Exec(schema); err != nil {
		log.Fatalf("failed to apply schema to %q: %v", path, err)
	}

	if err := migrateSchema(context.Background(), conn); err != nil {
		log.Fatalf("failed to migrate schema of %q: %v", path, err)
	}

	if err := seedPages(context.Background(), conn); err != nil {
		log.Fatalf("failed to seed pages into %q: %v", path, err)
	}

	fmt.Printf("Database ready at %s\n", path)
	return conn
}

// seedPagesJSON holds the legacy app's 51 searchable pages (archived Wikipedia articles),
// exported from its old database. go:embed packs the file into the binary at build time,
// so every environment (local, CI, the VM) gets the same pages without copying any files.
//
//go:embed seed/pages.json
var seedPagesJSON []byte

// seedPage is one row of seed/pages.json.
type seedPage struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

// seedPages inserts the legacy pages. INSERT OR IGNORE skips pages whose title or url
// already exists, so it is safe to run on every startup.
func seedPages(ctx context.Context, conn *sql.DB) error {
	var pages []seedPage
	if err := json.Unmarshal(seedPagesJSON, &pages); err != nil {
		return fmt.Errorf("parse seed pages: %w", err)
	}

	// One transaction, so the pages are written together instead of one disk write each
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	for _, p := range pages {
		_, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO pages (title, url, content, language) VALUES (?, ?, ?, ?)`,
			p.Title, p.URL, p.Content, p.Language,
		)
		if err != nil {
			return fmt.Errorf("insert seed page %q: %w", p.Title, err)
		}
	}
	return tx.Commit()
}
