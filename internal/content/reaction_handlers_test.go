// Package content provides tests for the forum content handlers and
// database operations.
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

const (
	reactionHandlerTestUserID    int64 = 1
	reactionHandlerTestPostID    int64 = 1
	reactionHandlerTestCommentID int64 = 1

	reactionHandlerTestSessionToken = "test-session-token"
)

func TestReactPostHandlerStoresRequestedReaction(t *testing.T) {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			handler := newReactionPostTestHandler(db)

			request := newAuthenticatedReactionRequest(
				http.MethodPost,
				test.path,
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertResponseStatus(
				t,
				recorder,
				http.StatusSeeOther,
			)

			assertRedirectLocation(
				t,
				recorder,
				"/posts/1",
			)

			assertStoredReaction(
				t,
				db,
				reactionHandlerTestUserID,
				reactionHandlerTestPostID,
				models.TargetPost,
				test.expectedValue,
			)
		})
	}
}

func TestReactCommentHandlerStoresRequestedReaction(t *testing.T) {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			handler := newReactionCommentTestHandler(db)

			request := newAuthenticatedReactionRequest(
				http.MethodPost,
				test.path,
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertResponseStatus(
				t,
				recorder,
				http.StatusSeeOther,
			)

			assertRedirectLocation(
				t,
				recorder,
				"/posts/1",
			)

			assertStoredReaction(
				t,
				db,
				reactionHandlerTestUserID,
				reactionHandlerTestCommentID,
				models.TargetComment,
				test.expectedValue,
			)
		})
	}
}

func TestReactPostHandlerRemovesRepeatedReaction(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionPostTestHandler(db)

	firstRequest := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/like",
	)
	firstRecorder := httptest.NewRecorder()

	handler.ServeHTTP(firstRecorder, firstRequest)

	assertResponseStatus(
		t,
		firstRecorder,
		http.StatusSeeOther,
	)

	secondRequest := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/like",
	)
	secondRecorder := httptest.NewRecorder()

	handler.ServeHTTP(secondRecorder, secondRequest)

	assertResponseStatus(
		t,
		secondRecorder,
		http.StatusSeeOther,
	)

	assertReactionDoesNotExist(
		t,
		db,
		reactionHandlerTestUserID,
		reactionHandlerTestPostID,
		models.TargetPost,
	)
}

func TestReactPostHandlerChangesLikeToDislike(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionPostTestHandler(db)

	likeRequest := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/like",
	)
	likeRecorder := httptest.NewRecorder()

	handler.ServeHTTP(likeRecorder, likeRequest)

	assertResponseStatus(
		t,
		likeRecorder,
		http.StatusSeeOther,
	)

	dislikeRequest := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/dislike",
	)
	dislikeRecorder := httptest.NewRecorder()

	handler.ServeHTTP(dislikeRecorder, dislikeRequest)

	assertResponseStatus(
		t,
		dislikeRecorder,
		http.StatusSeeOther,
	)

	assertStoredReaction(
		t,
		db,
		reactionHandlerTestUserID,
		reactionHandlerTestPostID,
		models.TargetPost,
		models.Dislike,
	)
}

func TestReactPostHandlerRejectsUnauthenticatedUser(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionPostTestHandler(db)

	request := httptest.NewRequest(
		http.MethodPost,
		"/posts/1/like",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertResponseStatus(
		t,
		recorder,
		http.StatusUnauthorized,
	)

	assertReactionDoesNotExist(
		t,
		db,
		reactionHandlerTestUserID,
		reactionHandlerTestPostID,
		models.TargetPost,
	)
}

func TestReactCommentHandlerRejectsUnauthenticatedUser(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionCommentTestHandler(db)

	request := httptest.NewRequest(
		http.MethodPost,
		"/comments/1/like",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertResponseStatus(
		t,
		recorder,
		http.StatusUnauthorized,
	)

	assertReactionDoesNotExist(
		t,
		db,
		reactionHandlerTestUserID,
		reactionHandlerTestCommentID,
		models.TargetComment,
	)
}

func TestReactPostHandlerRejectsInvalidPostID(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "non-numeric post ID",
			path: "/posts/not-a-number/like",
		},
		{
			name: "zero post ID",
			path: "/posts/0/like",
		},
		{
			name: "negative post ID",
			path: "/posts/-1/like",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			handler := newReactionPostTestHandler(db)

			request := newAuthenticatedReactionRequest(
				http.MethodPost,
				test.path,
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertResponseStatus(
				t,
				recorder,
				http.StatusBadRequest,
			)

			assertNoReactionsStored(t, db)
		})
	}
}

func TestReactCommentHandlerRejectsInvalidCommentID(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "non-numeric comment ID",
			path: "/comments/not-a-number/like",
		},
		{
			name: "zero comment ID",
			path: "/comments/0/like",
		},
		{
			name: "negative comment ID",
			path: "/comments/-1/like",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupReactionHandlerTestDB(t)
			seedReactionHandlerTestData(t, db)

			handler := newReactionCommentTestHandler(db)

			request := newAuthenticatedReactionRequest(
				http.MethodPost,
				test.path,
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assertResponseStatus(
				t,
				recorder,
				http.StatusBadRequest,
			)

			assertNoReactionsStored(t, db)
		})
	}
}

func TestReactPostHandlerReturnsInternalServerErrorWhenDatabaseFails(
	t *testing.T,
) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionPostTestHandler(db)

	if err := db.Close(); err != nil {
		t.Fatalf("close test database: %v", err)
	}

	request := newAuthenticatedReactionRequest(
		http.MethodPost,
		"/posts/1/like",
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertResponseStatus(
		t,
		recorder,
		http.StatusInternalServerError,
	)
}

func TestReactPostHandlerRejectsWrongHTTPMethod(t *testing.T) {
	db := setupReactionHandlerTestDB(t)
	seedReactionHandlerTestData(t, db)

	handler := newReactionPostTestHandler(db)

	request := newAuthenticatedReactionRequest(
		http.MethodGet,
		"/posts/1/like",
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertResponseStatus(
		t,
		recorder,
		http.StatusMethodNotAllowed,
	)

	assertNoReactionsStored(t, db)
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

	if _, err := db.Exec(reactionHandlerTestSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedReactionHandlerTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)

	insertReactionHandlerTestUser(t, db, now)
	insertReactionHandlerTestSession(t, db, now, expiresAt)
	insertReactionHandlerTestPost(t, db, now)
	insertReactionHandlerTestComment(t, db, now)
}

func insertReactionHandlerTestUser(
	t *testing.T,
	db *sql.DB,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO users (
			id,
			username,
			email,
			password_hash,
			created_at
		) VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestUserID,
		"test-user",
		"test@example.com",
		"test-password-hash",
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
}

func insertReactionHandlerTestSession(
	t *testing.T,
	db *sql.DB,
	createdAt time.Time,
	expiresAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO sessions (
			token,
			user_id,
			expires_at,
			created_at
		) VALUES (?, ?, ?, ?)`,
		reactionHandlerTestSessionToken,
		reactionHandlerTestUserID,
		expiresAt.Format(time.RFC3339),
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}
}

func insertReactionHandlerTestPost(
	t *testing.T,
	db *sql.DB,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO posts (
			id,
			user_id,
			title,
			body,
			created_at
		) VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestPostID,
		reactionHandlerTestUserID,
		"Test post",
		"Test post body",
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}
}

func insertReactionHandlerTestComment(
	t *testing.T,
	db *sql.DB,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO comments (
			id,
			post_id,
			user_id,
			body,
			created_at
		) VALUES (?, ?, ?, ?, ?)`,
		reactionHandlerTestCommentID,
		reactionHandlerTestPostID,
		reactionHandlerTestUserID,
		"Test comment",
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}
}

func newReactionPostTestHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.Handle(
		"POST /posts/{id}/like",
		ReactPostHandler(db),
	)

	mux.Handle(
		"POST /posts/{id}/dislike",
		ReactPostHandler(db),
	)

	return auth.WithUser(db)(
		auth.RequireAuth(mux),
	)
}

func newReactionCommentTestHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.Handle(
		"POST /comments/{id}/like",
		ReactCommentHandler(db),
	)

	mux.Handle(
		"POST /comments/{id}/dislike",
		ReactCommentHandler(db),
	)

	return auth.WithUser(db)(
		auth.RequireAuth(mux),
	)
}

func newAuthenticatedReactionRequest(
	method string,
	path string,
) *http.Request {
	request := httptest.NewRequest(
		method,
		path,
		nil,
	)

	request.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: reactionHandlerTestSessionToken,
	})

	return request
}

func assertResponseStatus(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedStatus int,
) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf(
			"response status = %d, want %d",
			recorder.Code,
			expectedStatus,
		)
	}
}

func assertRedirectLocation(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedLocation string,
) {
	t.Helper()

	location := recorder.Header().Get("Location")

	if location != expectedLocation {
		t.Errorf(
			"redirect location = %q, want %q",
			location,
			expectedLocation,
		)
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

	err := db.QueryRow(
		`SELECT value
		FROM reactions
		WHERE user_id = ?
		  AND target_id = ?
		  AND target_type = ?`,
		userID,
		targetID,
		targetType,
	).Scan(&storedValue)
	if err != nil {
		t.Fatalf("query stored reaction: %v", err)
	}

	if storedValue != expectedValue {
		t.Errorf(
			"stored reaction value = %d, want %d",
			storedValue,
			expectedValue,
		)
	}
}

func assertReactionDoesNotExist(
	t *testing.T,
	db *sql.DB,
	userID int64,
	targetID int64,
	targetType models.ReactionTarget,
) {
	t.Helper()

	var count int

	err := db.QueryRow(
		`SELECT COUNT(*)
		FROM reactions
		WHERE user_id = ?
		  AND target_id = ?
		  AND target_type = ?`,
		userID,
		targetID,
		targetType,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count stored reactions: %v", err)
	}

	if count != 0 {
		t.Errorf("stored reaction count = %d, want 0", count)
	}
}

func assertNoReactionsStored(t *testing.T, db *sql.DB) {
	t.Helper()

	var count int

	err := db.QueryRow(
		`SELECT COUNT(*) FROM reactions`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count reactions: %v", err)
	}

	if count != 0 {
		t.Errorf("stored reaction count = %d, want 0", count)
	}
}

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
	user_id    INTEGER NOT NULL UNIQUE
		REFERENCES users(id),
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE posts (
	id         INTEGER PRIMARY KEY,
	user_id    INTEGER NOT NULL
		REFERENCES users(id),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE comments (
	id         INTEGER PRIMARY KEY,
	post_id    INTEGER NOT NULL
		REFERENCES posts(id),
	user_id    INTEGER NOT NULL
		REFERENCES users(id),
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE reactions (
	id          INTEGER PRIMARY KEY,
	user_id     INTEGER NOT NULL
		REFERENCES users(id),
	target_id   INTEGER NOT NULL,
	target_type TEXT NOT NULL CHECK (
		target_type IN ('post', 'comment')
	),
	value       INTEGER NOT NULL CHECK (
		value IN (1, -1)
	),
	created_at  TEXT NOT NULL,
	UNIQUE (user_id, target_id, target_type)
);
`
