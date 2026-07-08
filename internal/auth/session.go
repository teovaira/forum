package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"

	"forum/internal/models"
)

// sessionDuration is how long a session stays valid after login.
const sessionDuration = 2 * time.Hour

// sessionCookieName is the cookie that carries the session token, per §4.7.
const sessionCookieName = "session_token"

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

// GetSessionUser resolves the user tied to the request's session cookie.
// A missing cookie, an unknown token, and an expired session are all
// treated as "not logged in" (nil user, nil error) rather than as errors —
// per §4.1, a session past its expires_at is ignored even though the row
// still physically exists in the table.
func GetSessionUser(db *sql.DB, r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, nil
	}

	row := db.QueryRow(`
		SELECT users.id, users.username, users.email, users.password_hash, users.created_at
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token = ? AND sessions.expires_at > ?`,
		cookie.Value, time.Now().Format(time.RFC3339),
	)

	var (
		user      models.User
		createdAt string
	)
	err = row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
