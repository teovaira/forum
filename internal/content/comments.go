package content

import (
	"database/sql"
	"errors"
)

// CreateComment creates a new comment on a post.
func CreateComment(db *sql.DB, postID, userID int64, body string) (int64, error) {
	return 0, errors.New("not implemented")
}
