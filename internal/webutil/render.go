// Package webutil holds the shared HTML-rendering and error-response helpers
// every handler in this project uses, so responses look and behave the same
// no matter which package produced them.
package webutil

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// templates holds the parsed set every RenderTemplate call executes against.
// It is package-level (not a RenderTemplate parameter) because the contract
// signature is frozen to (w, name, data); SetTemplates is how callers supply
// the parsed set, and tests substitute their own template set directly.
var templates *template.Template

// SetTemplates installs the template set RenderTemplate will execute
// against. cmd/server calls this once at startup with the real templates
// parsed from web/templates/*.html.
func SetTemplates(t *template.Template) {
	templates = t
}

// RenderTemplate executes the named template with data and writes the
// result to w. It renders into a buffer first so a template execution
// error never leaves a half-written page in the response.
func RenderTemplate(w http.ResponseWriter, name string, data any) error {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := buf.WriteTo(w)
	return err
}

// ErrorPageData is the view-data passed to error.html.
type ErrorPageData struct {
	StatusCode int
	Message    string
}

// RenderError writes statusCode and renders error.html with message. It
// renders into a buffer before writing anything, so if error.html itself
// fails to execute, the response falls back to a plain-text body instead of
// silently sending nothing — an error path is the one place a rendering
// failure must never go unnoticed.
func RenderError(w http.ResponseWriter, statusCode int, message string) {
	data := ErrorPageData{StatusCode: statusCode, Message: message}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "error.html", data); err != nil {
		log.Printf("webutil: failed to render error.html: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, "%d %s", statusCode, message)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	buf.WriteTo(w)
}
