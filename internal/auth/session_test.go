package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
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

// seedUser inserts a minimal user row directly and returns its ID, so
// session tests don't depend on the auth handlers (Stage B10) existing yet.
func seedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(
		"INSERT INTO users (username, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		"tester", "tester@example.com", "not-a-real-hash", time.Now().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read last insert id: %v", err)
	}
	return id
}

func TestGetSessionUser(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()
	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	userID := seedUser(t, db)

	t.Run("valid session returns the user", func(t *testing.T) {
		token, _, err := CreateSession(db, userID)
		if err != nil {
			t.Fatalf("CreateSession returned an error: %v", err)
		}

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session_token", Value: token})

		user, err := GetSessionUser(db, r)
		if err != nil {
			t.Fatalf("GetSessionUser returned an error: %v", err)
		}
		if user == nil {
			t.Fatal("expected a user, got nil")
		}
		if user.ID != userID {
			t.Errorf("user.ID = %d, want %d", user.ID, userID)
		}
	})

	t.Run("missing cookie returns no user, no error", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		user, err := GetSessionUser(db, r)
		if err != nil {
			t.Fatalf("expected no error for a missing cookie, got: %v", err)
		}
		if user != nil {
			t.Errorf("expected nil user for a missing cookie, got %+v", user)
		}
	})

	t.Run("expired session returns no user, no error", func(t *testing.T) {
		token, _, err := CreateSession(db, userID)
		if err != nil {
			t.Fatalf("CreateSession returned an error: %v", err)
		}
		// Force this session into the past directly, bypassing CreateSession's
		// fixed 2h duration, so the test can assert expiry is enforced.
		past := time.Now().Add(-time.Hour).Format(time.RFC3339)
		if _, err := db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?", past, token); err != nil {
			t.Fatalf("failed to backdate session: %v", err)
		}

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session_token", Value: token})

		user, err := GetSessionUser(db, r)
		if err != nil {
			t.Fatalf("expected no error for an expired session, got: %v", err)
		}
		if user != nil {
			t.Errorf("expected nil user for an expired session, got %+v", user)
		}
	})
}

func TestDestroySession(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()
	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	userID := seedUser(t, db)
	token, _, err := CreateSession(db, userID)
	if err != nil {
		t.Fatalf("CreateSession returned an error: %v", err)
	}

	if err := DestroySession(db, token); err != nil {
		t.Fatalf("DestroySession returned an error: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "session_token", Value: token})

	user, err := GetSessionUser(db, r)
	if err != nil {
		t.Fatalf("GetSessionUser returned an error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user after DestroySession, got %+v", user)
	}

	// Destroying a token that no longer exists must not be an error —
	// logout is idempotent.
	if err := DestroySession(db, token); err != nil {
		t.Errorf("destroying an already-destroyed token should not error, got: %v", err)
	}
}

func TestCreateSessionStoresTimestampsInUTC(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()
	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	token, _, err := CreateSession(db, seedUser(t, db))
	if err != nil {
		t.Fatalf("CreateSession returned an error: %v", err)
	}

	var expiresAt, createdAt string
	err = db.QueryRow(
		"SELECT expires_at, created_at FROM sessions WHERE token = ?", token,
	).Scan(&expiresAt, &createdAt)
	if err != nil {
		t.Fatalf("failed to read the session row: %v", err)
	}

	for _, column := range []struct{ name, value string }{
		{"expires_at", expiresAt},
		{"created_at", createdAt},
	} {
		if !strings.HasSuffix(column.value, "Z") {
			t.Errorf("%s = %q, want a UTC timestamp ending in Z", column.name, column.value)
		}
	}
}

func TestGetSessionUserAcceptsSessionStoredInUTC(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()
	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	// expires_at is compared as a plain SQL string, so a row written in UTC
	// must still resolve on a server whose local zone is not UTC.
	now := time.Now().UTC()
	_, err = db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		"utc-stored-token", seedUser(t, db),
		now.Add(time.Hour).Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to insert a UTC session: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "utc-stored-token"})

	user, err := GetSessionUser(db, r)
	if err != nil {
		t.Fatalf("GetSessionUser returned an error: %v", err)
	}
	if user == nil {
		t.Fatal("expected a user for a session expiring in one hour, got nil")
	}
}
