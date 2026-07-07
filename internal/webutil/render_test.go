package webutil

import (
	"html/template"
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
