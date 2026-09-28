package main

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// errUserExists is returned by createUser when the username or email is already registered.
var errUserExists = errors.New("username or email already taken")

// getUserByUsername looks up a single user by username. It returns (nil, nil)
// when no such user exists, so callers can distinguish "not found" from a
// real query error without checking sql.ErrNoRows themselves.
func getUserByUsername(username string) (*User, error) {
	var u User
	err := db.QueryRow(
		`SELECT id, username, password FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Password)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// createUser inserts a new user with an already hashed password. It relies on the
// UNIQUE constraints on username and email instead of checking first, so two
// simultaneous registrations for the same name can't both succeed.
func createUser(ctx context.Context, username, email string, hashedPassword []byte) error {
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`,
		username, email, hashedPassword,
	)

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return errUserExists
	}
	return err
}

func searchPages(ctx context.Context, q, language string) ([]SearchResult, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT title, url, content FROM pages WHERE language = ? AND content LIKE ?`,
		language, "%"+q+"%",
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("closing search rows: %v", err)
		}
	}()
	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Title, &r.URL, &r.Description); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
