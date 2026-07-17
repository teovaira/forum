package content

import (
	"database/sql"
	"errors"
)

// PostFilter defines the criteria for filtering posts.
type PostFilter struct {
	CategoryID      *int64
	CreatedByUserID *int64
	LikedByUserID   *int64
}

// CreatePost creates a new forum post with categories.
func CreatePost(db *sql.DB, userID int64, title, body string, categoryIDs []int64) (int64, error) {
	return 0, errors.New("not implemented")
}