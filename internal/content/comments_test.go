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

func TestListComments(t *testing.T) {
	db := newPostTestDB(t)
	userID1 := insertUser(t, db, "user1")
	userID2 := insertUser(t, db, "user2")
	categoryID := insertCategory(t, db, "General", "discussion")

	postID, err := CreatePost(db, userID1, "Title", "Body", []int64{categoryID})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	t.Run("returns empty slice when there are no comments", func(t *testing.T) {
		comments, err := ListComments(db, postID)
		if err != nil {
			t.Fatalf("ListComments() error = %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("ListComments() returned %d comments, want 0", len(comments))
		}
	})

	t.Run("returns all comments for a post sorted by creation asc", func(t *testing.T) {
		commentID1, err := CreateComment(db, postID, userID1, "First comment")
		if err != nil {
			t.Fatalf("failed to create comment 1: %v", err)
		}

		commentID2, err := CreateComment(db, postID, userID2, "Second comment")
		if err != nil {
			t.Fatalf("failed to create comment 2: %v", err)
		}

		comments, err := ListComments(db, postID)
		if err != nil {
			t.Fatalf("ListComments() error = %v", err)
		}

		if len(comments) != 2 {
			t.Fatalf("ListComments() returned %d comments, want 2", len(comments))
		}

		if comments[0].ID != commentID1 || comments[1].ID != commentID2 {
			t.Errorf("ListComments() order = [%d, %d], want [%d, %d]",
				comments[0].ID, comments[1].ID, commentID1, commentID2)
		}

		if comments[0].Author != "user1" || comments[1].Author != "user2" {
			t.Errorf("ListComments() authors = [%q, %q], want [%q, %q]",
				comments[0].Author, comments[1].Author, "user1", "user2")
		}
	})
}
