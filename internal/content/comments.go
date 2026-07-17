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
//
// It joins the comments table with the users table to fetch the author's username,
// orders comments by created_at ASC, and queries comment reactions using the
// CountReactions helper.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - postID: The ID of the post whose comments to retrieve.
//
// Returns:
//   - A slice of Comments matching the postID, or an empty slice if none exist.
//   - An error if any database operations fail.
func ListComments(db *sql.DB, postID int64) ([]models.Comment, error) {
	rows, err := db.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.body, c.created_at
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.post_id = ?
		 ORDER BY c.created_at ASC, c.id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	var commentIDs []int64

	for rows.Next() {
		var comment models.Comment
		var createdAtStr string
		err = rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Author, &comment.Body, &createdAtStr)
		if err != nil {
			return nil, err
		}
		comment.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
		commentIDs = append(commentIDs, comment.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return []models.Comment{}, nil
	}

	// Fetch reactions for all comments in batch
	reactionMap, err := CountReactions(db, models.TargetComment, commentIDs)
	if err != nil {
		return nil, err
	}

	// Populate reactions
	for i := range comments {
		id := comments[i].ID
		if counts, ok := reactionMap[id]; ok {
			comments[i].Likes = counts.Likes
			comments[i].Dislikes = counts.Dislikes
		}
	}

	return comments, nil
}
