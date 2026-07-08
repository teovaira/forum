package auth

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// sessionDuration is how long a session stays valid after login.
const sessionDuration = 2 * time.Hour

// CreateSession issues a new session for userID, replacing any session that
// user already holds. The sessions.user_id column is UNIQUE (schema.sql),
// so a plain INSERT would fail on a second login; deleting the old row
// first is what makes "log in again" replace the session instead of
// erroring, matching the spec's "only one opened session" requirement.
func CreateSession(db *sql.DB, userID int64) (token string, expiresAt time.Time, err error) {
	if _, err := db.Exec("DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return "", time.Time{}, err
	}

	token = uuid.NewString()
	expiresAt = time.Now().Add(sessionDuration)

	_, err = db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		token, userID, expiresAt.Format(time.RFC3339), time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}
