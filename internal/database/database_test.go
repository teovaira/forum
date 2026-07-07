package database

import "testing"

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
