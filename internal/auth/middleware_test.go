package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"forum/internal/database"
)

func TestWithUser(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect returned an error: %v", err)
	}
	defer db.Close()
	if err := database.Init(db); err != nil {
		t.Fatalf("Init returned an error: %v", err)
	}

	userID := seedUser(t, db)
	token, _, err := CreateSession(db, userID)
	if err != nil {
		t.Fatalf("CreateSession returned an error: %v", err)
	}

	t.Run("valid session populates context", func(t *testing.T) {
		var sawUser bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			sawUser = ok && user != nil && user.ID == userID
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()

		WithUser(db)(next).ServeHTTP(w, r)

		if !sawUser {
			t.Error("expected the wrapped handler to see the logged-in user in context")
		}
	})

	t.Run("no session still reaches the handler with no user", func(t *testing.T) {
		var called bool
		var hadUser bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			_, hadUser = UserFromContext(r.Context())
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		WithUser(db)(next).ServeHTTP(w, r)

		if !called {
			t.Fatal("expected WithUser to always call the wrapped handler")
		}
		if hadUser {
			t.Error("expected no user in context for a request with no session")
		}
	})
}
