package content

import (
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