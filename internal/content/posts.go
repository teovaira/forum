package content

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// PostFilter defines the criteria for filtering posts.
type PostFilter struct {
	CategoryID      *int64
	CreatedByUserID *int64
	LikedByUserID   *int64
}

// CreatePost creates a new forum post with categories.
//
// It validates that the title, body, and category list are not empty. It uses
// a database transaction to ensure both the post and its category relations
// are successfully inserted.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - userID: The ID of the author.
//   - title: The title of the post.
//   - body: The content body of the post.
//   - categoryIDs: A slice of category IDs to associate with this post.
//
// Returns:
//   - The newly created post's ID.
//   - An error if any database operation or validation fails.
func CreatePost(db *sql.DB, userID int64, title, body string, categoryIDs []int64) (int64, error) {
	trimmedTitle := strings.TrimSpace(title)
	trimmedBody := strings.TrimSpace(body)

	if trimmedTitle == "" {
		return 0, errors.New("title cannot be empty or whitespace only")
	}
	if trimmedBody == "" {
		return 0, errors.New("body cannot be empty or whitespace only")
	}
	if len(categoryIDs) == 0 {
		return 0, errors.New("at least one category must be selected")
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	createdAt := time.Now().UTC().Format(time.RFC3339)

	stmt, err := tx.Prepare("INSERT INTO posts (user_id, title, body, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(userID, trimmedTitle, trimmedBody, createdAt)
	if err != nil {
		return 0, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	pcStmt, err := tx.Prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return 0, err
	}
	defer pcStmt.Close()

	for _, categoryID := range categoryIDs {
		_, err = pcStmt.Exec(postID, categoryID)
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return postID, nil
}
