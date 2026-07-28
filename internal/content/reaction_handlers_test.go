package content

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/internal/auth"
	"forum/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

const (
	reactionHandlerTestUserID       int64 = 1
	reactionHandlerTestPostID       int64 = 1
	reactionHandlerTestCommentID    int64 = 1
	reactionHandlerTestSessionToken       = "test-session-token"
)

const reactionHandlerTestSchema = `
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
	user_id    INTEGER NOT NULL UNIQUE REFERENCES users(id),
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE posts (
	id         INTEGER PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
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
	value       INTEGER NOT NULL CHECK (value IN (1, -1)),
	created_at  TEXT NOT NULL,
	UNIQUE (user_id, target_id, target_type)
);
`

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

	if _, err := db.Exec(reactionHandlerTestSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedReactionHandlerTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)

	_, err := db.Exec(
		`INSERT INTO users (id, username, email, password_hash, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestUserID,
		"test-user",
		"test@example.com",
		"test-password-hash",
		now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at, created_at)
		 VALUES (?, ?, ?, ?)`,
		reactionHandlerTestSessionToken,
		reactionHandlerTestUserID,
		expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO posts (id, user_id, title, body, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestPostID,
		reactionHandlerTestUserID,
		"Test post",
		"Test post body",
		now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO comments (id, post_id, user_id, body, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestCommentID,
		reactionHandlerTestPostID,
		reactionHandlerTestUserID,
		"Test comment",
		now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}
}

func newReactionPostTestHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /posts/{id}/like", ReactPostHandler(db))
	mux.Handle("POST /posts/{id}/dislike", ReactPostHandler(db))
	return auth.WithUser(db)(auth.RequireAuth(mux))
}

func newReactionCommentTestHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /comments/{id}/like", ReactCommentHandler(db))
	mux.Handle("POST /comments/{id}/dislike", ReactCommentHandler(db))
	return auth.WithUser(db)(auth.RequireAuth(mux))
}

func newAuthenticatedReactionRequest(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: reactionHandlerTestSessionToken,
	})
	return r
}

func assertHandlerStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Errorf("status = %d, want %d", rec.Code, want)
	}
}

func assertHandlerRedirect(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	got := rec.Header().Get("Location")
	if got != want {
		t.Errorf("redirect location = %q, want %q", got, want)
	}
}

func assertHandlerStoredReaction(t *testing.T, db *sql.DB, userID, targetID int64, targetType models.ReactionTarget, want models.ReactionValue) {
	t.Helper()
	var got models.ReactionValue
	err := db.QueryRow(
		`SELECT value FROM reactions WHERE user_id=? AND target_id=? AND target_type=?`,
		userID, targetID, targetType,
	).Scan(&got)
	if err != nil {
		t.Fatalf("query stored reaction: %v", err)
	}
	if got != want {
		t.Errorf("stored reaction value = %d, want %d", got, want)
	}
}

func assertHandlerNoReaction(t *testing.T, db *sql.DB, userID, targetID int64, targetType models.ReactionTarget) {
	t.Helper()
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM reactions WHERE user_id=? AND target_id=? AND target_type=?`,
		userID, targetID, targetType,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count reactions: %v", err)
	}
	if count != 0 {
		t.Errorf("reaction count = %d, want 0", count)
	}
}

func assertHandlerNoReactionsStored(t *testing.T, db *sql.DB) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reactions`).Scan(&count); err != nil {
		t.Fatalf("count reactions: %v", err)
	}
	if count != 0 {
		t.Errorf("reaction count = %d, want 0", count)
	}
}

func TestReactPostHandlerStoresRequestedReaction(t *testing.T) {
	tests := []struct {
		name string
		path string
		want models.ReactionValue
	}{
		{"like post", "/posts/1/like", models.Like},
		{"dislike post", "/posts/1/dislike", models.Dislike},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			rec := httptest.NewRecorder()
			newReactionPostTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, test.path, ""))

			assertHandlerStatus(t, rec, http.StatusSeeOther)
			assertHandlerRedirect(t, rec, "/posts/1")
			assertHandlerStoredReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestPostID, models.TargetPost, test.want)
		})
	}
}

func TestReactCommentHandlerStoresRequestedReaction(t *testing.T) {
	tests := []struct {
		name string
		path string
		want models.ReactionValue
	}{
		{"like comment", "/comments/1/like", models.Like},
		{"dislike comment", "/comments/1/dislike", models.Dislike},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			rec := httptest.NewRecorder()
			newReactionCommentTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, test.path, "post_id=1"))

			assertHandlerStatus(t, rec, http.StatusSeeOther)
			assertHandlerRedirect(t, rec, "/posts/1")
			assertHandlerStoredReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestCommentID, models.TargetComment, test.want)
		})
	}
}

func TestReactPostHandlerRemovesRepeatedReaction(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	h := newReactionPostTestHandler(db)

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, newAuthenticatedReactionRequest(http.MethodPost, "/posts/1/like", ""))
	assertHandlerStatus(t, rec1, http.StatusSeeOther)

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, newAuthenticatedReactionRequest(http.MethodPost, "/posts/1/like", ""))
	assertHandlerStatus(t, rec2, http.StatusSeeOther)

	assertHandlerNoReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestPostID, models.TargetPost)
}

func TestReactPostHandlerChangesLikeToDislike(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	h := newReactionPostTestHandler(db)

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, newAuthenticatedReactionRequest(http.MethodPost, "/posts/1/like", ""))
	assertHandlerStatus(t, rec1, http.StatusSeeOther)

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, newAuthenticatedReactionRequest(http.MethodPost, "/posts/1/dislike", ""))
	assertHandlerStatus(t, rec2, http.StatusSeeOther)

	assertHandlerStoredReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestPostID, models.TargetPost, models.Dislike)
}

func TestReactPostHandlerRejectsUnauthenticatedUser(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	rec := httptest.NewRecorder()
	newReactionPostTestHandler(db).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/posts/1/like", nil))

	assertHandlerStatus(t, rec, http.StatusUnauthorized)
	assertHandlerNoReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestPostID, models.TargetPost)
}

func TestReactCommentHandlerRejectsUnauthenticatedUser(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	rec := httptest.NewRecorder()
	newReactionCommentTestHandler(db).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/comments/1/like", nil))

	assertHandlerStatus(t, rec, http.StatusUnauthorized)
	assertHandlerNoReaction(t, db, reactionHandlerTestUserID, reactionHandlerTestCommentID, models.TargetComment)
}

func TestReactPostHandlerRejectsInvalidPostID(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"non-numeric post ID", "/posts/not-a-number/like"},
		{"zero post ID", "/posts/0/like"},
		{"negative post ID", "/posts/-1/like"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			rec := httptest.NewRecorder()
			newReactionPostTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, test.path, ""))

			assertHandlerStatus(t, rec, http.StatusBadRequest)
			assertHandlerNoReactionsStored(t, db)
		})
	}
}

func TestReactCommentHandlerRejectsInvalidCommentID(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"non-numeric comment ID", "/comments/not-a-number/like"},
		{"zero comment ID", "/comments/0/like"},
		{"negative comment ID", "/comments/-1/like"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			rec := httptest.NewRecorder()
			newReactionCommentTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, test.path, "post_id=1"))

			assertHandlerStatus(t, rec, http.StatusBadRequest)
			assertHandlerNoReactionsStored(t, db)
		})
	}
}

func TestReactCommentHandlerRejectsMissingPostID(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	rec := httptest.NewRecorder()
	newReactionCommentTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, "/comments/1/like", ""))

	assertHandlerStatus(t, rec, http.StatusBadRequest)
}

func TestReactCommentHandlerRejectsInvalidPostID(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"non-numeric post ID", "post_id=not-a-number"},
		{"zero post ID", "post_id=0"},
		{"negative post ID", "post_id=-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			rec := httptest.NewRecorder()
			newReactionCommentTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodPost, "/comments/1/like", test.body))

			assertHandlerStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestReactPostHandlerReturnsInternalServerErrorWhenDatabaseFails(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	mux := http.NewServeMux()
	mux.Handle(
		"POST /posts/{id}/{reaction}",
		ReactPostHandler(db),
	)

	h := auth.WithUser(db)(
		auth.RequireAuth(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := db.Close(); err != nil {
					t.Fatalf("close test database: %v", err)
				}

				mux.ServeHTTP(w, r)
			}),
		),
	)

	rec := httptest.NewRecorder()
	req := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/like",
		"",
	)

	h.ServeHTTP(rec, req)

	assertHandlerStatus(t, rec, http.StatusInternalServerError)
}

func TestReactPostHandlerRejectsWrongHTTPMethod(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	rec := httptest.NewRecorder()
	newReactionPostTestHandler(db).ServeHTTP(rec, newAuthenticatedReactionRequest(http.MethodGet, "/posts/1/like", ""))

	assertHandlerStatus(t, rec, http.StatusMethodNotAllowed)
	assertHandlerNoReactionsStored(t, db)
}
