package main

import (
	"database/sql"
	"forum/internal/database"
	"forum/internal/webutil"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLookupEnv(t *testing.T) {
	t.Run("returns the fallback when the variable is unset", func(t *testing.T) {
		if got := lookupEnv("FORUM_TEST_UNSET", "8080"); got != "8080" {
			t.Errorf("lookupEnv() = %q, want %q", got, "8080")
		}
	})

	t.Run("returns the environment value when it is set", func(t *testing.T) {
		t.Setenv("FORUM_TEST_PORT", "9999")

		if got := lookupEnv("FORUM_TEST_PORT", "8080"); got != "9999" {
			t.Errorf("lookupEnv() = %q, want %q", got, "9999")
		}
	})

	// An exported but blank PORT would otherwise build the listen address
	// ":" and bind an arbitrary free port instead of the intended one.
	t.Run("returns the fallback when the variable is set but empty", func(t *testing.T) {
		t.Setenv("FORUM_TEST_EMPTY", "")

		if got := lookupEnv("FORUM_TEST_EMPTY", "8080"); got != "8080" {
			t.Errorf("lookupEnv() = %q, want %q", got, "8080")
		}
	})
}

func TestParseTemplates(t *testing.T) {
	dir := filepath.Join("..", "..", "web", "templates")

	t.Run("parses every page template in the directory", func(t *testing.T) {
		templates, err := parseTemplates(dir)
		if err != nil {
			t.Fatalf("parseTemplates() error = %v, want nil", err)
		}

		// webutil looks templates up by base filename, so every page the
		// handlers render must be registered under exactly that name.
		for _, name := range []string{
			"home.html", "post.html", "new_post.html",
			"login.html", "register.html", "error.html",
		} {
			if templates.Lookup(name) == nil {
				t.Errorf("template %q was not registered", name)
			}
		}
	})

	t.Run("registers the shared blocks layout.html defines", func(t *testing.T) {
		templates, err := parseTemplates(dir)
		if err != nil {
			t.Fatalf("parseTemplates() error = %v, want nil", err)
		}

		for _, name := range []string{"header", "simple-header", "footer"} {
			if templates.Lookup(name) == nil {
				t.Errorf("shared block %q was not registered", name)
			}
		}
	})

	t.Run("executes error.html by name, as RenderError does", func(t *testing.T) {
		templates, err := parseTemplates(dir)
		if err != nil {
			t.Fatalf("parseTemplates() error = %v, want nil", err)
		}

		data := struct {
			StatusCode int
			Message    string
		}{StatusCode: 404, Message: "Post not found."}

		if err := templates.ExecuteTemplate(io.Discard, "error.html", data); err != nil {
			t.Errorf("ExecuteTemplate(error.html) error = %v, want nil", err)
		}
	})

	t.Run("returns an error for a directory with no templates", func(t *testing.T) {
		if _, err := parseTemplates(t.TempDir()); err == nil {
			t.Error("parseTemplates() error = nil, want an error for an empty directory")
		}
	})
}

func TestBuildHandler(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("database.Connect() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Init(db); err != nil {
		t.Fatalf("database.Init() error = %v", err)
	}

	templates, err := parseTemplates(filepath.Join("..", "..", "web", "templates"))
	if err != nil {
		t.Fatalf("parseTemplates() error = %v", err)
	}
	webutil.SetTemplates(templates)

	handler := buildHandler(db, filepath.Join("..", "..", "web", "static"))

	get := func(target string) *http.Response {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec.Result()
	}

	t.Run("content routes are registered", func(t *testing.T) {
		if got := get("/").StatusCode; got == http.StatusNotFound {
			t.Errorf("GET / status = %d, want registered (not 404)", got)
		}
	})

	t.Run("auth routes are registered", func(t *testing.T) {
		if got := get("/login").StatusCode; got == http.StatusNotFound {
			t.Errorf("GET /login status = %d, want registered (not 404)", got)
		}
	})

	t.Run("static routes are registered", func(t *testing.T) {
		if got := get("/static/").StatusCode; got == http.StatusNotFound {
			t.Errorf("GET /static/ status = %d, want registered (not 404)", got)
		}
	})

	t.Run("method-aware patterns still 405 on a mismatch", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/login", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE /login status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	// content.RegisterRoutes wraps this route in auth.RequireAuth internally;
	// buildHandler must apply WithUser globally so RequireAuth has a context
	// to read. Without it, this would 401 exactly the same on the surface,
	// but for the wrong reason (context never populated at all) rather than
	// the right one (no session cookie present).
	t.Run("a protected route blocks a guest without panicking", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/new", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET /posts/new as guest status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()

	// Returns the db handle alongside the server so a test can assert
	// against the same rows the HTTP requests wrote.

	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("database.Connect() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Init(db); err != nil {
		t.Fatalf("database.Init() error = %v", err)
	}

	templates, err := parseTemplates(filepath.Join("..", "..", "web", "templates"))
	if err != nil {
		t.Fatalf("parseTemplates() error = %v", err)
	}
	webutil.SetTemplates(templates)

	server := httptest.NewServer(buildHandler(db, filepath.Join("..", "..", "web", "static")))
	t.Cleanup(server.Close)
	return server, db
}

func newTestClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	return &http.Client{
		Jar: jar,
		// Redirects are not followed so tests can assert on the redirect
		// itself rather than on its destination page.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func firstCategoryID(t *testing.T, server *httptest.Server, client *http.Client) int64 {
	t.Helper()

	// Scraped from the form rather than hardcoded, because the schema's
	// auto-increment does not guarantee any particular category ID.

	resp, err := client.Get(server.URL + "/posts/new")
	if err != nil {
		t.Fatalf("GET /posts/new error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /posts/new body: %v", err)
	}

	start := strings.Index(string(body), `name="categories" value="`)
	if start == -1 {
		t.Fatal("no category checkbox found on /posts/new")
	}
	start += len(`name="categories" value="`)
	end := strings.Index(string(body)[start:], `"`)
	if end == -1 {
		t.Fatal("malformed category checkbox on /posts/new")
	}

	id, err := strconv.ParseInt(string(body)[start:start+end], 10, 64)
	if err != nil {
		t.Fatalf("parse category id: %v", err)
	}
	return id
}

func TestEndToEndForumJourney(t *testing.T) {
	server, _ := newTestServer(t)
	client := newTestClient(t)

	t.Run("a guest can browse but not reach a protected page", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/")
		if err != nil {
			t.Fatalf("GET / error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		resp, err = client.Get(server.URL + "/posts/new")
		if err != nil {
			t.Fatalf("GET /posts/new error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET /posts/new as guest status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	t.Run("registering logs the user in", func(t *testing.T) {
		form := url.Values{
			"email":    {"journey@example.com"},
			"username": {"journeyuser"},
			"password": {"hunter2hunter2"},
		}
		resp, err := client.PostForm(server.URL+"/register", form)
		if err != nil {
			t.Fatalf("POST /register error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST /register status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
		}

		resp, err = client.Get(server.URL + "/posts/new")
		if err != nil {
			t.Fatalf("GET /posts/new error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /posts/new after registering status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	var postID string

	t.Run("a logged-in user can create a post with a category", func(t *testing.T) {
		categoryID := firstCategoryID(t, server, client)

		form := url.Values{
			"title":      {"My Journey Post"},
			"body":       {"Body of the journey post."},
			"categories": {strconv.FormatInt(categoryID, 10)},
		}
		resp, err := client.PostForm(server.URL+"/posts", form)
		if err != nil {
			t.Fatalf("POST /posts error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST /posts status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
		}

		location := resp.Header.Get("Location")
		if !strings.HasPrefix(location, "/posts/") {
			t.Fatalf("POST /posts redirected to %q, want a /posts/{id} location", location)
		}
		postID = strings.TrimPrefix(location, "/posts/")

		homeResp, err := client.Get(server.URL + "/")
		if err != nil {
			t.Fatalf("GET / error = %v", err)
		}
		defer homeResp.Body.Close()
		body, _ := io.ReadAll(homeResp.Body)
		if !strings.Contains(string(body), "My Journey Post") {
			t.Error("new post did not appear on the home page")
		}
	})

	t.Run("a logged-in user can comment on the post", func(t *testing.T) {
		form := url.Values{"body": {"A comment in the journey test."}}
		resp, err := client.PostForm(server.URL+"/posts/"+postID+"/comments", form)
		if err != nil {
			t.Fatalf("POST /posts/{id}/comments error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("POST /posts/{id}/comments status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
		}

		postResp, err := client.Get(server.URL + "/posts/" + postID)
		if err != nil {
			t.Fatalf("GET /posts/{id} error = %v", err)
		}
		defer postResp.Body.Close()
		body, _ := io.ReadAll(postResp.Body)
		if !strings.Contains(string(body), "A comment in the journey test.") {
			t.Error("new comment did not appear on the post page")
		}
	})

	t.Run("a logged-in user can like the post", func(t *testing.T) {
		resp, err := client.PostForm(server.URL+"/posts/"+postID+"/like", url.Values{})
		if err != nil {
			t.Fatalf("POST /posts/{id}/like error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("POST /posts/{id}/like status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
		}

		postResp, err := client.Get(server.URL + "/posts/" + postID)
		if err != nil {
			t.Fatalf("GET /posts/{id} error = %v", err)
		}
		defer postResp.Body.Close()
		body, _ := io.ReadAll(postResp.Body)
		// Logged-in users see the interactive reaction buttons, not the
		// guest-only "N likes / N dislikes" summary line, so the count is
		// asserted via the button label post.html renders for a member.
		if !strings.Contains(string(body), "Like (1)") {
			t.Error("like count did not update to 1 on the post page")
		}
	})

	t.Run("logging out blocks the protected route again", func(t *testing.T) {
		resp, err := client.PostForm(server.URL+"/logout", url.Values{})
		if err != nil {
			t.Fatalf("POST /logout error = %v", err)
		}
		resp.Body.Close()

		resp, err = client.Get(server.URL + "/posts/new")
		if err != nil {
			t.Fatalf("GET /posts/new error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET /posts/new after logout status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})
}

func TestEndToEndCategoryFilter(t *testing.T) {
	server, _ := newTestServer(t)
	client := newTestClient(t)

	form := url.Values{
		"email": {"filtertest@example.com"}, "username": {"filtertester"}, "password": {"hunter2hunter2"},
	}
	if resp, err := client.PostForm(server.URL+"/register", form); err != nil {
		t.Fatalf("POST /register error = %v", err)
	} else {
		resp.Body.Close()
	}

	categoryID := firstCategoryID(t, server, client)

	postForm := url.Values{
		"title": {"Categorized Post"}, "body": {"Body."},
		"categories": {strconv.FormatInt(categoryID, 10)},
	}
	resp, err := client.PostForm(server.URL+"/posts", postForm)
	if err != nil {
		t.Fatalf("POST /posts error = %v", err)
	}
	resp.Body.Close()

	filtered, err := client.Get(server.URL + "/?category=" + strconv.FormatInt(categoryID, 10))
	if err != nil {
		t.Fatalf("GET /?category=%d error = %v", categoryID, err)
	}
	defer filtered.Body.Close()
	body, _ := io.ReadAll(filtered.Body)
	if !strings.Contains(string(body), "Categorized Post") {
		t.Error("filtering by the post's own category did not show it")
	}
}

func TestEndToEndSchemaIsAuditable(t *testing.T) {
	server, db := newTestServer(t)
	client := newTestClient(t)

	form := url.Values{
		"email": {"audit@example.com"}, "username": {"audituser"}, "password": {"hunter2hunter2"},
	}
	if resp, err := client.PostForm(server.URL+"/register", form); err != nil {
		t.Fatalf("POST /register error = %v", err)
	} else {
		resp.Body.Close()
	}

	categoryID := firstCategoryID(t, server, client)
	postForm := url.Values{
		"title": {"Audit Post"}, "body": {"Audit body."},
		"categories": {strconv.FormatInt(categoryID, 10)},
	}
	resp, err := client.PostForm(server.URL+"/posts", postForm)
	if err != nil {
		t.Fatalf("POST /posts error = %v", err)
	}
	resp.Body.Close()
	postID := strings.TrimPrefix(resp.Header.Get("Location"), "/posts/")

	commentResp, err := client.PostForm(server.URL+"/posts/"+postID+"/comments", url.Values{"body": {"Audit comment."}})
	if err != nil {
		t.Fatalf("POST /posts/{id}/comments error = %v", err)
	}
	commentResp.Body.Close()

	// Queried directly against the shared db, the same way the audit opens
	// the .db file with sqlite3 and runs SELECT * FROM users/posts/comments.
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "audit@example.com").Scan(&userCount); err != nil {
		t.Fatalf("SELECT * FROM users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("users row count for the registered email = %d, want 1", userCount)
	}

	var postTitle string
	if err := db.QueryRow("SELECT title FROM posts WHERE id = ?", postID).Scan(&postTitle); err != nil {
		t.Fatalf("SELECT * FROM posts: %v", err)
	}
	if postTitle != "Audit Post" {
		t.Errorf("posts.title = %q, want %q", postTitle, "Audit Post")
	}

	var commentBody string
	if err := db.QueryRow("SELECT body FROM comments WHERE post_id = ?", postID).Scan(&commentBody); err != nil {
		t.Fatalf("SELECT * FROM comments: %v", err)
	}
	if commentBody != "Audit comment." {
		t.Errorf("comments.body = %q, want %q", commentBody, "Audit comment.")
	}
}
