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
