package main

import (
	"context"
	"database/sql"
	"errors"

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

const (
	maxSearchResults  = 50  // hard cap until real pagination exists
	descriptionMaxLen = 200 // characters of content shown as a preview
)

func searchPages(ctx context.Context, q, language string) (results []SearchResult, err error) {
	pattern := "%" + q + "%"

	var rows *sql.Rows
	rows, err = db.QueryContext(ctx,
		`SELECT title, url, substr(content, 1, ?) FROM pages
		 WHERE language = ? AND (title LIKE ? OR content LIKE ?)
		 LIMIT ?`,
		descriptionMaxLen, language, pattern, pattern, maxSearchResults,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	for rows.Next() {
		var r SearchResult
		if scanErr := rows.Scan(&r.Title, &r.URL, &r.Description); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
