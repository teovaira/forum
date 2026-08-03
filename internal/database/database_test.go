package database

import (
	"context"
	"testing"
)

func TestConnect(t *testing.T) {
	db, err := Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()

	var foreignKeysEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		t.Fatalf("failed to query foreign_keys pragma: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Errorf("expected foreign_keys = 1, got %d", foreignKeysEnabled)
	}
}

func TestConnectEnablesForeignKeysOnEveryPooledConnection(t *testing.T) {
	db, err := Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()

	// PRAGMA foreign_keys is a per-connection setting; force the pool to
	// open a second physical connection so this checks a connection the
	// initial PRAGMA in Connect never touched.
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)

	first, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire the first connection: %v", err)
	}
	defer first.Close()

	second, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire a second connection: %v", err)
	}
	defer second.Close()

	var foreignKeysEnabled int
	err = second.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeysEnabled)
	if err != nil {
		t.Fatalf("failed to query foreign_keys pragma: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Errorf("expected foreign_keys = 1 on a second pooled connection, got %d", foreignKeysEnabled)
	}
}

func TestInit(t *testing.T) {
	db, err := Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()

	if err := Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	var tableName string
	err = db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'users'",
	).Scan(&tableName)
	if err != nil {
		t.Fatalf("expected users table to exist after Init: %v", err)
	}

	if err := Init(db); err != nil {
		t.Fatalf("calling Init a second time should be safe, got error: %v", err)
	}
}
