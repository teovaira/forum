// Package webutil holds the shared HTML-rendering and error-response helpers
// every handler in this project uses, so responses look and behave the same
// no matter which package produced them.
package webutil

import (
	"bytes"
	"html/template"
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
