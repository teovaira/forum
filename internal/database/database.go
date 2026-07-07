package database

import (
	"database/sql"
	_ "embed"

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

//go:embed schema.sql
var schema string

// Init creates the database schema. It is safe to call on every server
// startup because schema.sql is written with idempotent statements
// (CREATE TABLE IF NOT EXISTS, INSERT OR IGNORE).
func Init(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
