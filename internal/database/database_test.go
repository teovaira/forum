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
