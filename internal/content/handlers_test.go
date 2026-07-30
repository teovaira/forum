package content

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"forum/internal/auth"
	"forum/internal/webutil"
)

func TestMain(m *testing.M) {
	// Initialize a dummy template set in webutil to prevent nil panics during content handler tests
	webutil.SetTemplates(template.Must(template.New("test").Parse(
		`{{define "home.html"}}home: {{len .Posts}} posts, ActiveFilter: {{.ActiveFilter}}{{if gt (len .Posts) 0}} (First post Likes: {{(index .Posts 0).Likes}}, Dislikes: {{(index .Posts 0).Dislikes}}){{end}}{{end}}` +
			`{{define "post.html"}}post: {{.Post.Title}} (Likes: {{.Post.Likes}}, Dislikes: {{.Post.Dislikes}}) - Comments: {{range .Comments}}{{.Body}} (Likes: {{.Likes}}, Dislikes: {{.Dislikes}}); {{end}} - {{.ErrorMessage}}{{end}}` +
			`{{define "new_post.html"}}new_post: {{.ErrorMessage}}{{end}}` +
			`{{define "error.html"}}error: {{.Message}}{{end}}`,
	)))
	os.Exit(m.Run())
}

func newAuthRequest(method, path string, body url.Values, db *sql.DB, username string) (*http.Request, string) {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}

	if username == "" {
		return r, ""
	}

	// Insert user
	var userID int64
	err := db.QueryRow(
		`INSERT INTO users (username, email, password_hash, created_at)
		 VALUES (?, ?, ?, ?) RETURNING id`,
		username, username+"@example.com", "hash", "2026-01-01T00:00:00Z",
	).Scan(&userID)
	if err != nil {
		panic(err)
	}

	// Create session
	token, _, err := auth.CreateSession(db, userID)
	if err != nil {
		panic(err)
	}

	r.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	return r, token
}

func TestHomeHandlerIntegration(t *testing.T) {
	db := newPostTestDB(t)
	_ = insertCategory(t, db, "Action", "genre")

	mux := http.NewServeMux()
	// Register the handler wrapped with WithUser global middleware
	mux.Handle("GET /", auth.WithUser(db)(HomeHandler(db)))

	t.Run("guest successfully views home feed", func(t *testing.T) {
		r, _ := newAuthRequest(http.MethodGet, "/", nil, db, "")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "home: 0 posts") {
			t.Errorf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("filters by category", func(t *testing.T) {
		r, _ := newAuthRequest(http.MethodGet, "/?category=1", nil, db, "")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "ActiveFilter: category:1") {
			t.Errorf("expected filter state to be present, body: %q", w.Body.String())
		}
	})

	t.Run("mine filter returns 401 for guests", func(t *testing.T) {
		r, _ := newAuthRequest(http.MethodGet, "/?filter=mine", nil, db, "")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("mine filter succeeds for logged in users", func(t *testing.T) {
		r, _ := newAuthRequest(http.MethodGet, "/?filter=mine", nil, db, "user1")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "ActiveFilter: mine") {
			t.Errorf("unexpected body: %q", w.Body.String())
		}
	})
}

func TestPostViewHandlerIntegration(t *testing.T) {
	db := newPostTestDB(t)
	categoryID := insertCategory(t, db, "Action", "genre")

	mux := http.NewServeMux()
	mux.Handle("GET /posts/{id}", auth.WithUser(db)(PostViewHandler(db)))

	t.Run("returns 404 for non-existent post", func(t *testing.T) {
		r, _ := newAuthRequest(http.MethodGet, "/posts/9999", nil, db, "")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("successfully renders existing post", func(t *testing.T) {
		// Create a post
		_, _ = newAuthRequest(http.MethodPost, "/", nil, db, "author")
		// Fetch userID from DB
		var userID int64
		_ = db.QueryRow("SELECT id FROM users WHERE username = 'author'").Scan(&userID)
		postID, _ := CreatePost(db, userID, "Post Title", "Post Body", []int64{categoryID})

		// Add a comment
		commentID, err := CreateComment(db, postID, userID, "Integration Comment")
		if err != nil {
			t.Fatalf("failed to create comment: %v", err)
		}

		// Insert post reactions
		_, err = db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, postID, "post", 1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert post like: %v", err)
		}

		otherUserID := insertUser(t, db, "otherauthor")
		_, err = db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			otherUserID, postID, "post", -1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert post dislike: %v", err)
		}

		// Insert comment reaction
		_, err = db.Exec(
			`INSERT INTO reactions (user_id, target_id, target_type, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, commentID, "comment", 1, "2026-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert comment like: %v", err)
		}

		r, _ := newAuthRequest(http.MethodGet, "/posts/"+strconv.FormatInt(postID, 10), nil, db, "")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}

		expectedOutput := "post: Post Title (Likes: 1, Dislikes: 1) - Comments: Integration Comment (Likes: 1, Dislikes: 0);"
		if !strings.Contains(w.Body.String(), expectedOutput) {
			t.Errorf("unexpected body: %q, wanted to contain: %q", w.Body.String(), expectedOutput)
		}
	})
}

func TestCreatePostHandlerIntegration(t *testing.T) {
	db := newPostTestDB(t)
	categoryID := insertCategory(t, db, "Action", "genre")

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	// Since RegisterRoutes uses auth.RequireAuth, we must wrap mux in WithUser middleware
	// so the session token cookie is evaluated before hitting RequireAuth.
	authenticatedMux := auth.WithUser(db)(mux)

	t.Run("redirects guest to login", func(t *testing.T) {
		form := url.Values{"title": {"Title"}, "body": {"Body"}, "categories": {strconv.FormatInt(categoryID, 10)}}
		r, _ := newAuthRequest(http.MethodPost, "/posts", form, db, "")
		w := httptest.NewRecorder()

		authenticatedMux.ServeHTTP(w, r)

		if w.Code != http.StatusSeeOther {
			t.Errorf("expected %d, got %d", http.StatusSeeOther, w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/login" {
			t.Errorf("expected redirect to /login, got %q", loc)
		}
	})

	t.Run("re-renders form on empty validation", func(t *testing.T) {
		form := url.Values{"title": {""}, "body": {""}, "categories": {strconv.FormatInt(categoryID, 10)}}
		r, _ := newAuthRequest(http.MethodPost, "/posts", form, db, "marios")
		w := httptest.NewRecorder()

		authenticatedMux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK (re-render), got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "new_post: Title and body are required fields.") {
			t.Errorf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("creates post and redirects on success", func(t *testing.T) {
		form := url.Values{"title": {"Valid Title"}, "body": {"Valid Body"}, "categories": {strconv.FormatInt(categoryID, 10)}}
		r, _ := newAuthRequest(http.MethodPost, "/posts", form, db, "marios_valid")
		w := httptest.NewRecorder()

		authenticatedMux.ServeHTTP(w, r)

		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303 Redirect, got %d", w.Code)
		}
		loc := w.Header().Get("Location")
		if !strings.HasPrefix(loc, "/posts/") {
			t.Errorf("unexpected redirect location: %q", loc)
		}
	})
}

func TestCreateCommentHandlerIntegration(t *testing.T) {
	db := newPostTestDB(t)
	categoryID := insertCategory(t, db, "Action", "genre")

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	authenticatedMux := auth.WithUser(db)(mux)

	// Create a post to comment on
	var userID int64
	_, _ = newAuthRequest(http.MethodPost, "/", nil, db, "author")
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'author'").Scan(&userID)
	postID, _ := CreatePost(db, userID, "Post Title", "Post Body", []int64{categoryID})

	postPath := fmt.Sprintf("/posts/%d/comments", postID)

	t.Run("validation failure re-renders post page with error", func(t *testing.T) {
		form := url.Values{"body": {""}}
		r, _ := newAuthRequest(http.MethodPost, postPath, form, db, "commenter")
		w := httptest.NewRecorder()

		authenticatedMux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK (re-render), got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Comment body cannot be empty.") {
			t.Errorf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("successfully comments and redirects", func(t *testing.T) {
		form := url.Values{"body": {"This is a valid comment!"}}
		r, _ := newAuthRequest(http.MethodPost, postPath, form, db, "commenter_valid")
		w := httptest.NewRecorder()

		authenticatedMux.ServeHTTP(w, r)

		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303 Redirect, got %d", w.Code)
		}
		loc := w.Header().Get("Location")
		expectedLoc := fmt.Sprintf("/posts/%d", postID)
		if loc != expectedLoc {
			t.Errorf("redirected to %q, want %q", loc, expectedLoc)
		}
	})
}
