package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of password. bcrypt generates a random
// salt per call, so hashing the same password twice yields different output
// — this is what stops two users with the same password from having
// identical rows in the database, and what makes precomputed rainbow-table
// attacks against the password column ineffective.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
