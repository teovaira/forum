package main

import (
	"forum/internal/database"
	"forum/internal/webutil"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
