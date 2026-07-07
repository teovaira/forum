package models

import "time"

// User is a registered forum member. PasswordHash never holds the raw
// password — only bcrypt's output — so a database leak does not expose
// plaintext credentials.
type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
