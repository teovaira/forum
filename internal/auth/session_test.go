package auth

import (
	"testing"
	"time"

	"forum/internal/database"
)

func TestCreateSession(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()

	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	const userID int64 = 1

	token1, expiresAt, err := CreateSession(db, userID)
	if err != nil {
		t.Fatalf("CreateSession returned an error: %v", err)
	}
	if token1 == "" {
		t.Fatal("CreateSession returned an empty token")
	}

	wantExpiry := time.Now().Add(2 * time.Hour)
	if expiresAt.Before(wantExpiry.Add(-time.Minute)) || expiresAt.After(wantExpiry.Add(time.Minute)) {
		t.Errorf("expiresAt = %v, want roughly %v (2h from now)", expiresAt, wantExpiry)
	}

	// A second session for the same user must replace the first — the
	// schema's UNIQUE(user_id) constraint allows only one row per user.
	token2, _, err := CreateSession(db, userID)
	if err != nil {
		t.Fatalf("second CreateSession returned an error: %v", err)
	}
	if token1 == token2 {
		t.Fatal("expected a new token on the second call")
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 session row for user %d, got %d", userID, count)
	}

	var tokenInDB string
	err = db.QueryRow("SELECT token FROM sessions WHERE user_id = ?", userID).Scan(&tokenInDB)
	if err != nil {
		t.Fatalf("failed to read token: %v", err)
	}
	if tokenInDB != token2 {
		t.Errorf("expected the surviving session to hold the second token, got %q", tokenInDB)
	}
}
