package content

import (
	"database/sql"
	"strings"
	"testing"
)

func TestCreatePost(t *testing.T) {
	t.Run("successfully creates a post with category", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")

		postID, err := CreatePost(db, userID, "My first post", "This is the body", []int64{categoryID})
		if err != nil {
			t.Fatalf("CreatePost() unexpected error = %v", err)
		}

		if postID <= 0 {
			t.Errorf("CreatePost() returned post ID %d, want a positive ID", postID)
		}

		// Verify row in database
		var title, body, createdAt string
		var dbUserID int64
		err = db.QueryRow("SELECT user_id, title, body, created_at FROM posts WHERE id = ?", postID).
			Scan(&dbUserID, &title, &body, &createdAt)
		if err != nil {
			t.Fatalf("failed to query created post: %v", err)
		}

		if dbUserID != userID {
			t.Errorf("post user_id = %d, want %d", dbUserID, userID)
		}
		if title != "My first post" {
			t.Errorf("post title = %q, want %q", title, "My first post")
		}
		if body != "This is the body" {
			t.Errorf("post body = %q, want %q", body, "This is the body")
		}
		if createdAt == "" {
			t.Error("post created_at is empty")
		}

		// Verify category mapping
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM post_categories WHERE post_id = ? AND category_id = ?", postID, categoryID).
			Scan(&count)
		if err != nil {
			t.Fatalf("failed to query post category mapping: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 mapping in post_categories, got %d", count)
		}
	})

	t.Run("rejects empty or whitespace-only title", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")

		_, err := CreatePost(db, userID, "   ", "Valid body", []int64{categoryID})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty title, got nil")
		}
	})

	t.Run("rejects empty or whitespace-only body", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")

		_, err := CreatePost(db, userID, "Valid Title", "", []int64{categoryID})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty body, got nil")
		}
	})

	t.Run("rejects empty categories list", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")

		_, err := CreatePost(db, userID, "Valid Title", "Valid body", []int64{})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty categories list, got nil")
		}
	})

	t.Run("enforces foreign key for user_id", func(t *testing.T) {
		db := newCategoryTestDB(t)
		categoryID := insertCategory(t, db, "General", "discussion")

		_, err := CreatePost(db, 999_999, "Valid Title", "Valid body", []int64{categoryID})
		if err == nil {
			t.Fatal("CreatePost() expected error for non-existent user_id, got nil")
		}
		if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "foreign key") {
			t.Errorf("CreatePost() expected foreign key error, got: %v", err)
		}
	})

	t.Run("enforces foreign key for category_id", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")

		_, err := CreatePost(db, userID, "Valid Title", "Valid body", []int64{999_999})
		if err == nil {
			t.Fatal("CreatePost() expected error for non-existent category_id, got nil")
		}
		if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "foreign key") {
			t.Errorf("CreatePost() expected foreign key error, got: %v", err)
		}
	})
}

func TestGetPost(t *testing.T) {
	t.Run("successfully retrieves an existing post with categories", func(t *testing.T) {
		db := newCategoryTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID1 := insertCategory(t, db, "Action", "genre")
		categoryID2 := insertCategory(t, db, "School", "theme")

		postID, err := CreatePost(db, userID, "Title", "Body", []int64{categoryID1, categoryID2})
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}

		post, err := GetPost(db, postID)
		if err != nil {
			t.Fatalf("GetPost() unexpected error = %v", err)
		}

		if post.ID != postID {
			t.Errorf("post.ID = %d, want %d", post.ID, postID)
		}
		if post.UserID != userID {
			t.Errorf("post.UserID = %d, want %d", post.UserID, userID)
		}
		if post.Author != "marios" {
			t.Errorf("post.Author = %q, want %q", post.Author, "marios")
		}
		if post.Title != "Title" {
			t.Errorf("post.Title = %q, want %q", post.Title, "Title")
		}
		if post.Body != "Body" {
			t.Errorf("post.Body = %q, want %q", post.Body, "Body")
		}
		if len(post.Categories) != 2 {
			t.Fatalf("len(post.Categories) = %d, want 2", len(post.Categories))
		}

		// Verify category names
		names := []string{post.Categories[0].Name, post.Categories[1].Name}
		hasAction := false
		hasSchool := false
		for _, name := range names {
			if name == "Action" {
				hasAction = true
			}
			if name == "School" {
				hasSchool = true
			}
		}
		if !hasAction || !hasSchool {
			t.Errorf("post categories = %v, want both Action and School", names)
		}
	})

	t.Run("returns ErrNoRows for non-existent post", func(t *testing.T) {
		db := newCategoryTestDB(t)
		_, err := GetPost(db, 999_999)
		if err != sql.ErrNoRows {
			t.Errorf("GetPost() error = %v, want sql.ErrNoRows", err)
		}
	})
}
