package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/database"
)

func newRegisterRequest(form url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestRegisterHandler(t *testing.T) {
	newDB := func(t *testing.T) *sql.DB {
		t.Helper()
		db, err := database.Connect(":memory:")
		if err != nil {
			t.Fatalf("Connect returned an error: %v", err)
		}
		t.Cleanup(func() { db.Close() })
		if err := database.Init(db); err != nil {
			t.Fatalf("Init returned an error: %v", err)
		}
		return db
	}

	t.Run("valid registration creates a user and sets a session cookie", func(t *testing.T) {
		db := newDB(t)
		form := url.Values{
			"email":    {"new@example.com"},
			"username": {"newuser"},
			"password": {"hunter2hunter2"},
		}
		w := httptest.NewRecorder()

		RegisterHandler(db)(w, newRegisterRequest(form))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("unexpected 500, body: %s", w.Body.String())
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "new@example.com").Scan(&count); err != nil {
			t.Fatalf("failed to count users: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected the user to be created, got count = %d", count)
		}

		var gotCookie bool
		for _, c := range w.Result().Cookies() {
			if c.Name == "session_token" && c.Value != "" {
				gotCookie = true
			}
		}
		if !gotCookie {
			t.Error("expected a session_token cookie to be set after registration")
		}
	})

	t.Run("empty fields are rejected, not silently inserted", func(t *testing.T) {
		cases := []url.Values{
			{"email": {""}, "username": {"u"}, "password": {"p"}},
			{"email": {"e@example.com"}, "username": {""}, "password": {"p"}},
			{"email": {"e@example.com"}, "username": {"u"}, "password": {""}},
			{"email": {"e@example.com"}, "username": {"u"}, "password": {"   "}},
		}

		for _, form := range cases {
			db := newDB(t)
			w := httptest.NewRecorder()

			RegisterHandler(db)(w, newRegisterRequest(form))

			if w.Code == http.StatusInternalServerError {
				t.Errorf("empty-field case should not 500, body: %s", w.Body.String())
			}

			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
				t.Fatalf("failed to count users: %v", err)
			}
			if count != 0 {
				t.Errorf("expected no user row for form %v, got count = %d", form, count)
			}
		}
	})

	t.Run("duplicate email is rejected with a friendly error, not a raw constraint error", func(t *testing.T) {
		db := newDB(t)
		form := url.Values{
			"email":    {"dupe@example.com"},
			"username": {"first"},
			"password": {"hunter2hunter2"},
		}
		RegisterHandler(db)(httptest.NewRecorder(), newRegisterRequest(form))

		form2 := url.Values{
			"email":    {"dupe@example.com"},
			"username": {"second"},
			"password": {"hunter2hunter2"},
		}
		w := httptest.NewRecorder()
		RegisterHandler(db)(w, newRegisterRequest(form2))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("duplicate email should not surface as a raw 500, body: %s", w.Body.String())
		}
		if strings.Contains(strings.ToUpper(w.Body.String()), "UNIQUE CONSTRAINT") {
			t.Error("raw SQLite constraint error leaked to the response body")
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "dupe@example.com").Scan(&count); err != nil {
			t.Fatalf("failed to count users: %v", err)
		}
		if count != 1 {
			t.Errorf("expected exactly 1 user with that email, got %d", count)
		}
	})
}
