package content

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"forum/internal/models"
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

// GetPost retrieves a single post by ID.
//
// It joins the posts table with the users table to fetch the author's username,
// loads the post's categories, and queries its reactions count using the
// CountReactions helper.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - postID: The ID of the post to retrieve.
//
// Returns:
//   - The Post matching the ID, or nil if not found.
//   - sql.ErrNoRows if no post matches postID, or a database error on failure.
func GetPost(db *sql.DB, postID int64) (*models.Post, error) {
	var post models.Post
	var createdAtStr string

	err := db.QueryRow(
		`SELECT p.id, p.user_id, u.username, p.title, p.body, p.created_at
		 FROM posts p
		 JOIN users u ON p.user_id = u.id
		 WHERE p.id = ?`,
		postID,
	).Scan(&post.ID, &post.UserID, &post.Author, &post.Title, &post.Body, &createdAtStr)

	if err != nil {
		return nil, err
	}

	post.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	// Fetch categories
	rows, err := db.Query(
		`SELECT c.id, c.name, c.kind
		 FROM post_categories pc
		 JOIN categories c ON pc.category_id = c.id
		 WHERE pc.post_id = ?`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Kind); err != nil {
			return nil, err
		}
		post.Categories = append(post.Categories, cat)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Fetch reactions
	reactionMap, err := CountReactions(db, models.TargetPost, []int64{postID})
	if err != nil {
		return nil, err
	}

	if counts, ok := reactionMap[postID]; ok {
		post.Likes = counts.Likes
		post.Dislikes = counts.Dislikes
	}

	return &post, nil
}

// ListPosts retrieves a list of posts matching the filter.
//
// It queries posts and joins with the users table. Categories and reactions are
// fetched in batch to optimize performance. Posts are ordered by created_at DESC.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - filter: The criteria to filter the post list.
//
// Returns:
//   - A slice of Posts matching the filter, or empty slice if none match.
//   - An error if any database operations fail.
func ListPosts(db *sql.DB, filter PostFilter) ([]models.Post, error) {
	var query string
	var args []any
	var conditions []string

	// Base query
	query = `SELECT p.id, p.user_id, u.username, p.title, p.body, p.created_at
			 FROM posts p
			 JOIN users u ON p.user_id = u.id`

	if filter.CreatedByUserID != nil {
		conditions = append(conditions, "p.user_id = ?")
		args = append(args, *filter.CreatedByUserID)
	}

	if filter.CategoryID != nil {
		postIDs, err := PostIDsInCategory(db, *filter.CategoryID)
		if err != nil {
			return nil, err
		}
		if len(postIDs) == 0 {
			return []models.Post{}, nil
		}
		// Construct IN clause
		var placeholders []string
		for _, id := range postIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		conditions = append(conditions, "p.id IN ("+strings.Join(placeholders, ", ")+")")
	}

	if filter.LikedByUserID != nil {
		postIDs, err := PostIDsLikedByUser(db, *filter.LikedByUserID)
		if err != nil {
			return nil, err
		}
		if len(postIDs) == 0 {
			return []models.Post{}, nil
		}
		// Construct IN clause
		var placeholders []string
		for _, id := range postIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		conditions = append(conditions, "p.id IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY p.created_at DESC, p.id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	var postIDs []int64

	for rows.Next() {
		var post models.Post
		var createdAtStr string
		err = rows.Scan(&post.ID, &post.UserID, &post.Author, &post.Title, &post.Body, &createdAtStr)
		if err != nil {
			return nil, err
		}
		post.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
		postIDs = append(postIDs, post.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return []models.Post{}, nil
	}

	// Fetch categories for all returned posts in batch
	var pcQuery string
	var pcArgs []any
	var pcPlaceholders []string
	for _, id := range postIDs {
		pcPlaceholders = append(pcPlaceholders, "?")
		pcArgs = append(pcArgs, id)
	}
	pcQuery = "SELECT pc.post_id, c.id, c.name, c.kind FROM post_categories pc JOIN categories c ON pc.category_id = c.id WHERE pc.post_id IN (" + strings.Join(pcPlaceholders, ", ") + ")"

	pcRows, err := db.Query(pcQuery, pcArgs...)
	if err != nil {
		return nil, err
	}
	defer pcRows.Close()

	postCatMap := make(map[int64][]models.Category)
	for pcRows.Next() {
		var postID int64
		var cat models.Category
		if err := pcRows.Scan(&postID, &cat.ID, &cat.Name, &cat.Kind); err != nil {
			return nil, err
		}
		postCatMap[postID] = append(postCatMap[postID], cat)
	}
	if err = pcRows.Err(); err != nil {
		return nil, err
	}

	// Fetch reactions for all returned posts in batch
	reactionMap, err := CountReactions(db, models.TargetPost, postIDs)
	if err != nil {
		return nil, err
	}

	// Populate the categories and reaction counts on posts
	for i := range posts {
		id := posts[i].ID
		posts[i].Categories = postCatMap[id]
		if counts, ok := reactionMap[id]; ok {
			posts[i].Likes = counts.Likes
			posts[i].Dislikes = counts.Dislikes
		}
	}

	return posts, nil
}
