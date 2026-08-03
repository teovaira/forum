package webutil

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	templates = template.Must(template.New("test").Parse(
		`{{define "greeting.html"}}Hello, {{.}}!{{end}}`,
	))

	w := httptest.NewRecorder()

	if err := RenderTemplate(w, "greeting.html", "World"); err != nil {
		t.Fatalf("RenderTemplate returned an error: %v", err)
	}

	got := w.Body.String()
	want := "Hello, World!"
	if got != want {
		t.Errorf("body = %q, want %q", got, want)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want prefix %q", ct, "text/html")
	}
}

func TestRenderTemplateMissingTemplate(t *testing.T) {
	templates = template.Must(template.New("test").Parse(
		`{{define "known.html"}}known{{end}}`,
	))

	w := httptest.NewRecorder()

	if err := RenderTemplate(w, "missing.html", nil); err == nil {
		t.Fatal("expected an error for an undefined template, got nil")
	}

	if n := w.Body.Len(); n != 0 {
		t.Errorf("expected nothing written to the response on error, got %d bytes", n)
	}
}

func TestSetTemplates(t *testing.T) {
	SetTemplates(template.Must(template.New("test").Parse(
		`{{define "set.html"}}installed via SetTemplates{{end}}`,
	)))

	w := httptest.NewRecorder()

	if err := RenderTemplate(w, "set.html", nil); err != nil {
		t.Fatalf("RenderTemplate returned an error: %v", err)
	}

	want := "installed via SetTemplates"
	if got := w.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestRenderError(t *testing.T) {
	templates = template.Must(template.New("test").Parse(
		`{{define "error.html"}}Error {{.StatusCode}}: {{.Message}}{{end}}`,
	))

	w := httptest.NewRecorder()

	RenderError(w, http.StatusNotFound, "Post not found.")

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}

	got := w.Body.String()
	want := "Error 404: Post not found."
	if got != want {
		t.Errorf("body = %q, want %q", got, want)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want prefix %q", ct, "text/html")
	}
}

func TestRenderErrorTemplateFailure(t *testing.T) {
	// No "error.html" defined here on purpose, so ExecuteTemplate fails
	// and RenderError must fall back to a plain-text body.
	templates = template.Must(template.New("test").Parse(
		`{{define "unrelated.html"}}x{{end}}`,
	))

	w := httptest.NewRecorder()

	RenderError(w, http.StatusInternalServerError, "boom")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	want := "500 boom"
	if got := w.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want prefix %q", ct, "text/plain")
	}
}
