package auth

import (
	"context"
	"database/sql"
	"net/http"

	"forum/internal/models"
	"forum/internal/webutil"
)

// contextKey is an unexported type so keys defined here can never collide
// with a context key from another package — a plain string key like
// "user" could clash with someone else's "user" key; a private named type
// cannot, because no other package can construct a value of this type.
type contextKey int

// userContextKey is the single key WithUser writes to and UserFromContext
// reads from. It is unexported so the only way into or out of the request
// context is through this package's own functions.
const userContextKey contextKey = 0

// WithUser returns middleware that looks up the request's session and, if
// valid, stores the user in the request context for downstream handlers.
// It never blocks the request — a guest (no or invalid session) still
// reaches next with no user in context; only RequireAuth enforces login.
func WithUser(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := GetSessionUser(db, r)
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext returns the logged-in user stored by WithUser, if any.
// The ok result distinguishes "no user in context" from a nil *models.User,
// mirroring the comma-ok idiom used for map lookups and type assertions.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
}

// RequireAuth blocks requests that have no user in context. Unlike
// WithUser, which only populates context and never blocks, this is the
// enforcement layer — wrap it around routes that must not be reachable by
// guests. A blocked request gets a clean 401 via webutil.RenderError,
// never a bare/raw status.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			webutil.RenderError(w, http.StatusUnauthorized, "You must be logged in to view this page.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
