package main

import "database/sql"

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
