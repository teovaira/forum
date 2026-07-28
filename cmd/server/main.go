// Package main is the forum server's entry point. It wires together
// internal/database, internal/auth, and internal/content, then starts
// listening for HTTP requests.
package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"forum/internal/auth"
	"forum/internal/content"
	"forum/internal/webutil"
)

// lookupEnv returns the value of the environment variable named key, falling
// back to fallback when that variable is unset or empty. An empty value is
// treated as unset because an exported but blank PORT would otherwise build
// the listen address ":", binding an arbitrary free port instead of 8080.
//
// Parameters:
//   - key: the name of the environment variable to read.
//   - fallback: the value to use when key is unset or empty.
//
// Returns:
//   - string: the environment value when it is set and non-empty, otherwise
//     fallback.
func lookupEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// parseTemplates parses every .html file in dir into a single template set.
// They must be parsed together, not individually, because the page templates
// invoke the shared blocks layout.html defines; a per-file parse would leave
// those blocks undefined at execution time.
//
// ParseGlob registers each file under its base name ("home.html"), which is
// the lookup key webutil.RenderTemplate and RenderError use.
//
// Parameters:
//   - dir: the directory holding the .html templates, e.g. "web/templates".
//
// Returns:
//   - *template.Template: the parsed set, ready for webutil.SetTemplates.
//   - error: non-nil if dir contains no .html files or one fails to parse.
func parseTemplates(dir string) (*template.Template, error) {
	return template.ParseGlob(filepath.Join(dir, "*.html"))
}

// buildHandler assembles the full request-handling pipeline: every package's
// routes on one mux, wrapped in auth.WithUser so a request's session is
// resolved before any handler runs.
//
// content.RegisterRoutes already wraps its own protected routes in
// auth.RequireAuth, so WithUser must be the only middleware applied here —
// adding RequireAuth again at this layer would block every route, since a
// route that isn't behind RequireAuth would suddenly require a session too.
//
// Parameters:
//   - db: an open connection pool, passed through to every package's routes.
//   - staticDir: the directory served at /static/, e.g. "web/static".
//
// Returns:
//   - http.Handler: the complete handler for http.ListenAndServe.
func buildHandler(db *sql.DB, staticDir string) http.Handler {
	mux := http.NewServeMux()

	auth.RegisterRoutes(mux, db)
	content.RegisterRoutes(mux, db)
	webutil.RegisterRoutes(mux, staticDir)

	return auth.WithUser(db)(mux)
}
