package content

import (
	"database/sql"
	"strings"
	"testing"

	"forum/internal/database"
)

const postTestSchema = `
PRAGMA foreign_keys = ON;

CREATE TABLE users (
	id            INTEGER PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TEXT NOT NULL
);

CREATE TABLE sessions (
	token      TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) UNIQUE,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL
);

CREATE TABLE categories (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	kind TEXT NOT NULL CHECK (
		kind IN ('demographic', 'genre', 'theme', 'discussion')
	)
);

CREATE TABLE posts (
	id         INTEGER PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE post_categories (
	post_id     INTEGER NOT NULL REFERENCES posts(id),
	category_id INTEGER NOT NULL REFERENCES categories(id),
	PRIMARY KEY (post_id, category_id)
);

CREATE TABLE comments (
	id         INTEGER PRIMARY KEY,
	post_id    INTEGER NOT NULL REFERENCES posts(id),
	user_id    INTEGER NOT NULL REFERENCES users(id),
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE reactions (
	id          INTEGER PRIMARY KEY,
	user_id     INTEGER NOT NULL REFERENCES users(id),
	target_id   INTEGER NOT NULL,
	target_type TEXT NOT NULL CHECK (target_type IN ('post', 'comment')),
	value       INTEGER NOT NULL CHECK (value = 1 OR value = -1),
	created_at  TEXT NOT NULL,
	UNIQUE (user_id, target_id, target_type)
);
`

func newPostTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("database.Connect() error = %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(postTestSchema); err != nil {
		t.Fatalf("create post test schema: %v", err)
	}

	return db
}

func TestCreatePost(t *testing.T) {
	t.Run("successfully creates a post with category", func(t *testing.T) {
		db := newPostTestDB(t)
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
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")

		_, err := CreatePost(db, userID, "   ", "Valid body", []int64{categoryID})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty title, got nil")
		}
	})

	t.Run("rejects empty or whitespace-only body", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")
		categoryID := insertCategory(t, db, "General", "discussion")

		_, err := CreatePost(db, userID, "Valid Title", "", []int64{categoryID})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty body, got nil")
		}
	})

	t.Run("rejects empty categories list", func(t *testing.T) {
		db := newPostTestDB(t)
		userID := insertUser(t, db, "marios")

		_, err := CreatePost(db, userID, "Valid Title", "Valid body", []int64{})
		if err == nil {
			t.Fatal("CreatePost() expected error for empty categories list, got nil")
		}
	})

	t.Run("enforces foreign key for user_id", func(t *testing.T) {
		db := newPostTestDB(t)
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
		db := newPostTestDB(t)
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
		db := newPostTestDB(t)
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

		// Verify reaction counts are populated correctly
		_, err = db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, postID, "post", 1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert like reaction: %v", err)
		}

		otherUserID := insertUser(t, db, "otheruser")
		_, err = db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			otherUserID, postID, "post", -1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert dislike reaction: %v", err)
		}

		postWithReactions, err := GetPost(db, postID)
		if err != nil {
			t.Fatalf("GetPost() with reactions unexpected error = %v", err)
		}

		if postWithReactions.Likes != 1 {
			t.Errorf("post.Likes = %d, want 1", postWithReactions.Likes)
		}
		if postWithReactions.Dislikes != 1 {
			t.Errorf("post.Dislikes = %d, want 1", postWithReactions.Dislikes)
		}
	})


	t.Run("returns ErrNoRows for non-existent post", func(t *testing.T) {
		db := newPostTestDB(t)
		_, err := GetPost(db, 999_999)
		if err != sql.ErrNoRows {
			t.Errorf("GetPost() error = %v, want sql.ErrNoRows", err)
		}
	})
}

func TestListPosts(t *testing.T) {
	db := newPostTestDB(t)
	userID1 := insertUser(t, db, "user1")
	userID2 := insertUser(t, db, "user2")

	categoryID1 := insertCategory(t, db, "Action", "genre")
	categoryID2 := insertCategory(t, db, "Romance", "genre")

	// Create posts
	postID1, err := CreatePost(db, userID1, "Post 1", "Body 1", []int64{categoryID1})
	if err != nil {
		t.Fatalf("failed to create post 1: %v", err)
	}

	postID2, err := CreatePost(db, userID2, "Post 2", "Body 2", []int64{categoryID2})
	if err != nil {
		t.Fatalf("failed to create post 2: %v", err)
	}

	postID3, err := CreatePost(db, userID1, "Post 3", "Body 3", []int64{categoryID1, categoryID2})
	if err != nil {
		t.Fatalf("failed to create post 3: %v", err)
	}

	t.Run("lists all posts sorted by creation desc", func(t *testing.T) {
		posts, err := ListPosts(db, PostFilter{})
		if err != nil {
			t.Fatalf("ListPosts() error = %v", err)
		}

		if len(posts) != 3 {
			t.Fatalf("ListPosts() returned %d posts, want 3", len(posts))
		}

		// Check order (newest first, which is post3, then post2, then post1)
		if posts[0].ID != postID3 || posts[1].ID != postID2 || posts[2].ID != postID1 {
			t.Errorf("ListPosts() order = [%d, %d, %d], want [%d, %d, %d]",
				posts[0].ID, posts[1].ID, posts[2].ID, postID3, postID2, postID1)
		}
	})

	t.Run("filters by category", func(t *testing.T) {
		posts, err := ListPosts(db, PostFilter{CategoryID: &categoryID1})
		if err != nil {
			t.Fatalf("ListPosts() error = %v", err)
		}

		if len(posts) != 2 {
			t.Fatalf("ListPosts() filtered by category returned %d posts, want 2", len(posts))
		}

		// Should be postID3 and postID1
		if posts[0].ID != postID3 || posts[1].ID != postID1 {
			t.Errorf("ListPosts() by category = [%d, %d], want [%d, %d]",
				posts[0].ID, posts[1].ID, postID3, postID1)
		}
	})

	t.Run("filters by author", func(t *testing.T) {
		posts, err := ListPosts(db, PostFilter{CreatedByUserID: &userID1})
		if err != nil {
			t.Fatalf("ListPosts() error = %v", err)
		}

		if len(posts) != 2 {
			t.Fatalf("ListPosts() filtered by author returned %d posts, want 2", len(posts))
		}

		// Should be postID3 and postID1
		if posts[0].ID != postID3 || posts[1].ID != postID1 {
			t.Errorf("ListPosts() by author = [%d, %d], want [%d, %d]",
				posts[0].ID, posts[1].ID, postID3, postID1)
		}
	})

	t.Run("filters by liked posts (when none are liked)", func(t *testing.T) {
		posts, err := ListPosts(db, PostFilter{LikedByUserID: &userID1})
		if err != nil {
			t.Fatalf("ListPosts() error = %v", err)
		}

		if len(posts) != 0 {
			t.Errorf("ListPosts() filtered by liked returned %d posts, want 0", len(posts))
		}
	})

	t.Run("filters by liked posts (when some are liked)", func(t *testing.T) {
		_, err := db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID1, postID2, "post", 1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert like reaction: %v", err)
		}

		posts, err := ListPosts(db, PostFilter{LikedByUserID: &userID1})
		if err != nil {
			t.Fatalf("ListPosts() error = %v", err)
		}

		if len(posts) != 1 {
			t.Fatalf("ListPosts() filtered by liked returned %d posts, want 1", len(posts))
		}

		if posts[0].ID != postID2 {
			t.Errorf("posts[0].ID = %d, want %d", posts[0].ID, postID2)
		}

		if posts[0].Likes != 1 {
			t.Errorf("posts[0].Likes = %d, want 1", posts[0].Likes)
		}
	})
}
