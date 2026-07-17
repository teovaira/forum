package content

import (
	"strings"
	"testing"
)

func TestCreateComment(t *testing.T) {
	t.Run("successfully creates a comment", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")
		postID, err := CreatePost(db, userID, "Title", "Body", []int64{categoryID})
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}

		commentID, err := CreateComment(db, postID, userID, "My first comment")
		if err != nil {
			t.Fatalf("CreateComment() unexpected error = %v", err)
		}

		if commentID <= 0 {
			t.Errorf("CreateComment() returned comment ID %d, want a positive ID", commentID)
		}

		// Verify comment in database
		var dbPostID, dbUserID int64
		var body, createdAt string
		err = db.QueryRow("SELECT post_id, user_id, body, created_at FROM comments WHERE id = ?", commentID).
			Scan(&dbPostID, &dbUserID, &body, &createdAt)
		if err != nil {
			t.Fatalf("failed to query created comment: %v", err)
		}

		if dbPostID != postID {
			t.Errorf("comment post_id = %d, want %d", dbPostID, postID)
		}
		if dbUserID != userID {
			t.Errorf("comment user_id = %d, want %d", dbUserID, userID)
		}
		if body != "My first comment" {
			t.Errorf("comment body = %q, want %q", body, "My first comment")
		}
		if createdAt == "" {
			t.Error("comment created_at is empty")
		}
	})

	t.Run("rejects empty or whitespace-only body", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")
		postID, err := CreatePost(db, userID, "Title", "Body", []int64{categoryID})
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}

		_, err = CreateComment(db, postID, userID, "   ")
		if err == nil {
			t.Fatal("CreateComment() expected error for empty body, got nil")
		}
	})

	t.Run("enforces foreign key for post_id", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")

		_, err := CreateComment(db, 999_999, userID, "Valid comment")
		if err == nil {
			t.Fatal("CreateComment() expected error for non-existent post_id, got nil")
		}
		if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "foreign key") {
			t.Errorf("CreateComment() expected foreign key error, got: %v", err)
		}
	})

	t.Run("enforces foreign key for user_id", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")
		postID, err := CreatePost(db, userID, "Title", "Body", []int64{categoryID})
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}

		_, err = CreateComment(db, postID, 999_999, "Valid comment")
		if err == nil {
			t.Fatal("CreateComment() expected error for non-existent user_id, got nil")
		}
		if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "foreign key") {
			t.Errorf("CreateComment() expected foreign key error, got: %v", err)
		}
	})
}
