package webutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{color:red}"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, dir)

	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	want := "body{color:red}"
	if got := w.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestRegisterRoutesMissingFile(t *testing.T) {
	dir := t.TempDir()

	mux := http.NewServeMux()
	RegisterRoutes(mux, dir)

	req := httptest.NewRequest(http.MethodGet, "/static/missing.css", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
