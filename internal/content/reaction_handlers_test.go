package content

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forum/internal/auth"
	"forum/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

func TestReactPostHandler(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedValue models.ReactionValue
	}{
		{
			name:          "like post",
			path:          "/posts/1/like",
			expectedValue: models.Like,
		},
		{
			name:          "dislike post",
			path:          "/posts/1/dislike",
			expectedValue: models.Dislike,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			mux := http.NewServeMux()

			mux.Handle(
				"POST /posts/{id}/like",
				ReactPostHandler(db),
			)

			mux.Handle(
				"POST /posts/{id}/dislike",
				ReactPostHandler(db),
			)

			handler := auth.WithUser(db)(
				auth.RequireAuth(mux),
			)

			request := httptest.NewRequest(
				http.MethodPost,
				tt.path,
				nil,
			)

			request.AddCookie(&http.Cookie{
				Name:  "session_token",
				Value: "test-session-token",
			})

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertStoredReaction(
				t,
				db,
				1,
				1,
				models.TargetPost,
				tt.expectedValue,
			)
		})
	}
}

func TestReactCommentHandler(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedValue models.ReactionValue
	}{
		{
			name:          "like comment",
			path:          "/comments/1/like",
			expectedValue: models.Like,
		},
		{
			name:          "dislike comment",
			path:          "/comments/1/dislike",
			expectedValue: models.Dislike,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			mux := http.NewServeMux()

			mux.Handle(
				"POST /comments/{id}/like",
				ReactCommentHandler(db),
			)

			mux.Handle(
				"POST /comments/{id}/dislike",
				ReactCommentHandler(db),
			)

			handler := auth.WithUser(db)(
				auth.RequireAuth(mux),
			)

			request := httptest.NewRequest(
				http.MethodPost,
				tt.path,
				nil,
			)

			request.AddCookie(&http.Cookie{
				Name:  "session_token",
				Value: "test-session-token",
			})

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertStoredReaction(
				t,
				db,
				1,
				1,
				models.TargetComment,
				tt.expectedValue,
			)
		})
	}
}

func setupReactionHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE TABLE comments (
			id INTEGER PRIMARY KEY,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			body TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE TABLE reactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			target_type TEXT NOT NULL
				CHECK(target_type IN ('post', 'comment')),
			target_id INTEGER NOT NULL,
			value INTEGER NOT NULL
				CHECK(value IN (1, -1)),
			created_at TEXT NOT NULL,
			UNIQUE(user_id, target_type, target_id)
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedReactionHandlerTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now().Format(time.RFC3339)
	expiresAt := time.Now().Add(time.Hour).Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO users (
			id,
			username,
			email,
			password_hash,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		1,
		"test-user",
		"test@example.com",
		"test-password-hash",
		now,
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sessions (
			token,
			user_id,
			expires_at,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		"test-session-token",
		1,
		expiresAt,
		now,
	)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO posts (
			id,
			user_id,
			title,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		1,
		1,
		"Test post",
		"Test post body",
		now,
	)
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO comments (
			id,
			post_id,
			user_id,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		1,
		1,
		1,
		"Test comment",
		now,
	)
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}
}

func assertStoredReaction(
	t *testing.T,
	db *sql.DB,
	userID int64,
	targetID int64,
	targetType models.ReactionTarget,
	expectedValue models.ReactionValue,
) {
	t.Helper()

	var storedValue models.ReactionValue

	err := db.QueryRow(`
		SELECT value
		FROM reactions
		WHERE user_id = ?
		  AND target_id = ?
		  AND target_type = ?
	`,
		userID,
		targetID,
		targetType,
	).Scan(&storedValue)
	if err != nil {
		t.Fatalf("query stored reaction: %v", err)
	}

	if storedValue != expectedValue {
		t.Errorf(
			"expected reaction value %d, got %d",
			expectedValue,
			storedValue,
		)
	}
}
