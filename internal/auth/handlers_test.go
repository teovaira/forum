package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"

	"forum/internal/database"
)

func newFormRequest(path string, form url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func newTestDB(t *testing.T) *sql.DB {
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

func TestRegisterHandler(t *testing.T) {
	t.Run("valid registration creates a user and sets a session cookie", func(t *testing.T) {
		db := newTestDB(t)
		form := url.Values{
			"email":    {"new@example.com"},
			"username": {"newuser"},
			"password": {"hunter2hunter2"},
		}
		w := httptest.NewRecorder()

		RegisterHandler(db)(w, newFormRequest("/register", form))

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
			db := newTestDB(t)
			w := httptest.NewRecorder()

			RegisterHandler(db)(w, newFormRequest("/register", form))

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
		db := newTestDB(t)
		form := url.Values{
			"email":    {"dupe@example.com"},
			"username": {"first"},
			"password": {"hunter2hunter2"},
		}
		RegisterHandler(db)(httptest.NewRecorder(), newFormRequest("/register", form))

		form2 := url.Values{
			"email":    {"dupe@example.com"},
			"username": {"second"},
			"password": {"hunter2hunter2"},
		}
		w := httptest.NewRecorder()
		RegisterHandler(db)(w, newFormRequest("/register", form2))

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

func TestLoginHandler(t *testing.T) {
	registerUser := func(t *testing.T, db *sql.DB, email, password string) {
		t.Helper()
		form := url.Values{"email": {email}, "username": {"someone"}, "password": {password}}
		RegisterHandler(db)(httptest.NewRecorder(), newFormRequest("/register", form))
	}

	t.Run("correct credentials log the user in", func(t *testing.T) {
		db := newTestDB(t)
		registerUser(t, db, "user@example.com", "correct-password")

		form := url.Values{"email": {"user@example.com"}, "password": {"correct-password"}}
		w := httptest.NewRecorder()

		LoginHandler(db)(w, newFormRequest("/login", form))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("unexpected 500, body: %s", w.Body.String())
		}

		var gotCookie bool
		for _, c := range w.Result().Cookies() {
			if c.Name == "session_token" && c.Value != "" {
				gotCookie = true
			}
		}
		if !gotCookie {
			t.Error("expected a session_token cookie to be set after a successful login")
		}
	})

	t.Run("wrong password is rejected with a friendly error", func(t *testing.T) {
		db := newTestDB(t)
		registerUser(t, db, "user2@example.com", "correct-password")

		form := url.Values{"email": {"user2@example.com"}, "password": {"wrong-password"}}
		w := httptest.NewRecorder()

		LoginHandler(db)(w, newFormRequest("/login", form))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("wrong password should not surface as a raw 500, body: %s", w.Body.String())
		}
		for _, c := range w.Result().Cookies() {
			if c.Name == "session_token" && c.Value != "" {
				t.Error("expected no session cookie to be set after a failed login")
			}
		}
	})

	t.Run("unknown email is rejected the same way as a wrong password", func(t *testing.T) {
		db := newTestDB(t)

		form := url.Values{"email": {"nobody@example.com"}, "password": {"whatever"}}
		w := httptest.NewRecorder()

		LoginHandler(db)(w, newFormRequest("/login", form))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("unknown email should not surface as a raw 500, body: %s", w.Body.String())
		}
	})

	t.Run("missing credentials show a warning, not a 500", func(t *testing.T) {
		db := newTestDB(t)

		form := url.Values{"email": {""}, "password": {""}}
		w := httptest.NewRecorder()

		LoginHandler(db)(w, newFormRequest("/login", form))

		if w.Code == http.StatusInternalServerError {
			t.Fatalf("missing credentials should not surface as a raw 500, body: %s", w.Body.String())
		}
	})
}

func TestLogoutHandler(t *testing.T) {
	db := newTestDB(t)
	form := url.Values{"email": {"logout@example.com"}, "username": {"someone"}, "password": {"hunter2hunter2"}}
	regW := httptest.NewRecorder()
	RegisterHandler(db)(regW, newFormRequest("/register", form))

	var token string
	for _, c := range regW.Result().Cookies() {
		if c.Name == "session_token" {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("setup failed: no session_token cookie after registration")
	}

	r := httptest.NewRequest(http.MethodPost, "/logout", nil)
	r.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	LogoutHandler(db)(w, r)

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count); err != nil {
		t.Fatalf("failed to count sessions: %v", err)
	}
	if count != 0 {
		t.Error("expected the session row to be deleted after logout")
	}

	var clearedCookie bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_token" && (c.MaxAge < 0 || c.Value == "") {
			clearedCookie = true
		}
	}
	if !clearedCookie {
		t.Error("expected the session_token cookie to be cleared on logout")
	}
}

func TestLogoutHandlerWithNoSession(t *testing.T) {
	// A guest (no session_token cookie at all) hitting /logout directly
	// must not panic or error — it should behave the same as logging out
	// an already-expired session: clear the cookie and redirect.
	db := newTestDB(t)

	r := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	LogoutHandler(db)(w, r)

	if w.Code == http.StatusInternalServerError {
		t.Fatalf("logout with no session should not 500, body: %s", w.Body.String())
	}

	var clearedCookie bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_token" && (c.MaxAge < 0 || c.Value == "") {
			clearedCookie = true
		}
	}
	if !clearedCookie {
		t.Error("expected the session_token cookie to still be cleared even with no prior session")
	}
}

func TestRegisterRoutes(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()

	RegisterRoutes(mux, db)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/register"},
		{http.MethodPost, "/register"},
		{http.MethodGet, "/login"},
		{http.MethodPost, "/login"},
		{http.MethodPost, "/logout"},
	}

	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s: route not registered (404)", c.method, c.path)
		}
	}

	// A method the pattern doesn't allow must 405, not fall through to a
	// handler (audit: "does the server use the right HTTP method?").
	r := httptest.NewRequest(http.MethodDelete, "/register", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /register: got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestIsUniqueConstraintError(t *testing.T) {
	uniqueErr := sqlite3.Error{
		Code:         sqlite3.ErrConstraint,
		ExtendedCode: sqlite3.ErrConstraintUnique,
	}
	if !isUniqueConstraintError(uniqueErr) {
		t.Error("expected true for a sqlite3 UNIQUE constraint error")
	}

	otherSQLiteErr := sqlite3.Error{
		Code:         sqlite3.ErrConstraint,
		ExtendedCode: sqlite3.ErrConstraintNotNull,
	}
	if isUniqueConstraintError(otherSQLiteErr) {
		t.Error("expected false for a non-UNIQUE sqlite3 constraint error")
	}

	// A plain error whose text merely contains the words must be judged by
	// type, not by string matching — this is the case the old string-based
	// check got wrong.
	if isUniqueConstraintError(errors.New("unique constraint failed: users.email")) {
		t.Error("expected false for a non-sqlite3 error, even if its text mentions the words")
	}

	if isUniqueConstraintError(nil) {
		t.Error("expected false for a nil error")
	}
}
