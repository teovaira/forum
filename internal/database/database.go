// Package database owns the SQLite connection and schema setup shared by
// every other package in this project.
package database

import (
	"database/sql"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database at path and enables foreign key
// enforcement, which SQLite otherwise leaves off per connection by default.
//
// The flag is set in the DSN, not via a PRAGMA issued after opening,
// because foreign_keys is a per-connection setting: a *sql.DB is a pool
// that can open more than one physical connection under concurrency, and a
// PRAGMA run once would only cover whichever connection happened to run it.
// The driver applies a DSN flag to every connection it opens, so
// enforcement holds regardless of pool size.
//
// Parameters:
//   - path: the SQLite DSN — a filesystem path (e.g. "./forum.db") or
//     ":memory:" for a temporary in-memory database.
//
// Returns:
//   - *sql.DB: a ready-to-use connection pool with foreign keys enabled on
//     every connection it opens.
//   - error: non-nil if the driver failed to open the database.
func Connect(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	return db, nil
}

//go:embed schema.sql
var schema string

// Init creates the database schema. It is safe to call on every server
// startup because schema.sql is written with idempotent statements
// (CREATE TABLE IF NOT EXISTS, INSERT OR IGNORE).
//
// Parameters:
//   - db: an open connection pool, typically the result of Connect.
//
// Returns:
//   - error: non-nil if executing schema.sql failed.
func Init(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
