// Package main is the forum server's entry point. It wires together
// internal/database, internal/auth, and internal/content, then starts
// listening for HTTP requests.
package main

import (
	"html/template"
	"os"
	"path/filepath"
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
