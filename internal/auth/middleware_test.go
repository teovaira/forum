package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"forum/internal/database"
	"forum/internal/models"
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

func TestRequireAuth(t *testing.T) {
	t.Run("user in context passes through", func(t *testing.T) {
		var called bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

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

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()

		// RequireAuth only checks context; WithUser is what populates it.
		WithUser(db)(RequireAuth(next)).ServeHTTP(w, r)

		if !called {
			t.Error("expected RequireAuth to call next when a user is present")
		}
	})

	t.Run("no user in context is blocked with a clean status", func(t *testing.T) {
		var called bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		RequireAuth(next).ServeHTTP(w, r)

		if called {
			t.Error("expected RequireAuth to block the request, but next was called")
		}

		if w.Code != http.StatusSeeOther {
			t.Errorf("expected a %d redirect, got %d", http.StatusSeeOther, w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/login" {
			t.Errorf("expected redirect to /login, got %q", loc)
		}
	})
}

func TestUserFromContext(t *testing.T) {
	t.Run("returns the stored user", func(t *testing.T) {
		want := &models.User{ID: 42, Username: "tester"}
		ctx := context.WithValue(context.Background(), userContextKey, want)

		got, ok := UserFromContext(ctx)

		if !ok {
			t.Fatal("expected ok = true for a context with a user stored")
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("empty context returns nil, false", func(t *testing.T) {
		user, ok := UserFromContext(context.Background())

		if ok {
			t.Error("expected ok = false for an empty context")
		}
		if user != nil {
			t.Errorf("expected a nil user, got %+v", user)
		}
	})
}
