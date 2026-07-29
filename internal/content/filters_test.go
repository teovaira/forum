package content

import (
	"database/sql"
	"testing"
)

func TestPostCategories(t *testing.T) {
	db := newPostTestDB(t)
	userID := insertUser(t, db, "marios")
	catID1 := insertCategory(t, db, "Action", "genre")
	catID2 := insertCategory(t, db, "Romance", "genre")

	// Create a post with two categories
	postID1, err := CreatePost(db, userID, "Post 1", "Body 1", []int64{catID1, catID2})
	if err != nil {
		t.Fatalf("failed to create post 1: %v", err)
	}

	// Create a post with one category
	postID2, err := CreatePost(db, userID, "Post 2", "Body 2", []int64{catID1})
	if err != nil {
		t.Fatalf("failed to create post 2: %v", err)
	}

	// Create a post with no category mapping manually
	var postID3 int64
	err = db.QueryRow(
		`INSERT INTO posts (user_id, title, body, created_at)
		 VALUES (?, ?, ?, ?) RETURNING id`,
		userID, "Post 3", "Body 3", "2026-01-01T00:00:00Z",
	).Scan(&postID3)
	if err != nil {
		t.Fatalf("failed to insert post 3: %v", err)
	}

	t.Run("successfully retrieves categories for batch", func(t *testing.T) {
		catMap, err := PostCategories(db, []int64{postID1, postID2, postID3})
		if err != nil {
			t.Fatalf("PostCategories() error = %v, want nil", err)
		}

		if len(catMap) != 3 {
			t.Errorf("len(catMap) = %d, want 3", len(catMap))
		}

		// Verify postID1 categories
		cats1 := catMap[postID1]
		if len(cats1) != 2 {
			t.Fatalf("expected 2 categories for post 1, got %d", len(cats1))
		}
		names1 := []string{cats1[0].Name, cats1[1].Name}
		if !(names1[0] == "Action" && names1[1] == "Romance" || names1[0] == "Romance" && names1[1] == "Action") {
			t.Errorf("post 1 categories = %v, want Action and Romance", names1)
		}

		// Verify postID2 categories
		cats2 := catMap[postID2]
		if len(cats2) != 1 {
			t.Fatalf("expected 1 category for post 2, got %d", len(cats2))
		}
		if cats2[0].Name != "Action" {
			t.Errorf("post 2 category = %q, want Action", cats2[0].Name)
		}

		// Verify postID3 categories
		cats3 := catMap[postID3]
		if len(cats3) != 0 {
			t.Errorf("expected 0 categories for post 3, got %d", len(cats3))
		}
	})

	t.Run("returns empty map for empty IDs slice", func(t *testing.T) {
		catMap, err := PostCategories(db, []int64{})
		if err != nil {
			t.Fatalf("PostCategories() error = %v, want nil", err)
		}
		if len(catMap) != 0 {
			t.Errorf("len(catMap) = %d, want 0", len(catMap))
		}
	})

	t.Run("returns query error when database is closed", func(t *testing.T) {
		closedDB, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("failed to open dummy db: %v", err)
		}
		closedDB.Close()

		_, err = PostCategories(closedDB, []int64{1})
		if err == nil {
			t.Error("expected error for closed database, got nil")
		}
	})
}
