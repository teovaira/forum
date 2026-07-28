package main

import (
	"io"
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
