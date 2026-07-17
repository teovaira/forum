package content

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"forum/internal/models"
)

// CreateComment creates a new comment on a post.
//
// It validates that the comment body is not empty or whitespace-only. On success,
// it inserts the comment into the database and returns the generated comment ID.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - postID: The ID of the post being commented on.
//   - userID: The ID of the author of the comment.
//   - body: The content of the comment.
//
// Returns:
//   - The newly created comment's ID.
//   - An error if validation or database operations fail.
func CreateComment(db *sql.DB, postID, userID int64, body string) (int64, error) {
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return 0, errors.New("comment body cannot be empty or whitespace only")
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)

	stmt, err := db.Prepare("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(postID, userID, trimmedBody, createdAt)
	if err != nil {
		return 0, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return commentID, nil
}

// ListComments retrieves all comments for a post.
func ListComments(db *sql.DB, postID int64) ([]models.Comment, error) {
	return nil, errors.New("not implemented")
}
