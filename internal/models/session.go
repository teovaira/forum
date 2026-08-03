package models

import "time"

// Session is a logged-in user's cookie-backed session row. Token is the
// value stored in the session_token cookie; ExpiresAt is checked on every
// request so a stale row is treated as "no user" even before it is deleted.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}
