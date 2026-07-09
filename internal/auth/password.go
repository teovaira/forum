// Package auth implements password hashing, cookie-based sessions, and the
// register/login/logout HTTP handlers for the forum.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of password. bcrypt generates a random
// salt per call, so hashing the same password twice yields different output
// — this is what stops two users with the same password from having
// identical rows in the database, and what makes precomputed rainbow-table
// attacks against the password column ineffective.
//
// Parameters:
//   - password: the plaintext password to hash.
//
// Returns:
//   - string: the bcrypt hash, safe to store in the database.
//   - error: non-nil if bcrypt failed to generate the hash.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches hash. It collapses
// bcrypt's error into a bool because every failure mode (wrong password,
// malformed hash) means the same thing to a caller: authentication did not
// succeed — the login handler never needs to know or leak which.
//
// Parameters:
//   - hash: a bcrypt hash previously produced by HashPassword.
//   - password: the plaintext password to check against hash.
//
// Returns:
//   - bool: true if password matches hash, false otherwise.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
