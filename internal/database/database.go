package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database at path and enables foreign key
// enforcement, which SQLite otherwise leaves off per connection by default.
func Connect(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
