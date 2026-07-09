package auth

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"forum/internal/models"
	"forum/internal/webutil"
)

// AuthPageData is the view-data contract for the register and login pages
// (§4.4). ErrorMessage is empty on first load; a failed submission re-renders
// the same page with a friendly message instead of a raw error.
type AuthPageData struct {
	ErrorMessage string
}

// baseSessionCookie returns a Cookie pre-filled with the attributes frozen
// in §4.7 — Name, Path, HttpOnly, SameSite=Strict, and the conditional
// Secure flag — so setSessionCookie and clearSessionCookie only need to
// fill in what actually differs between "set" and "clear" (Value/Expires
// vs. MaxAge), instead of each repeating the full attribute list.
func baseSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("SECURE_COOKIES") == "true",
		SameSite: http.SameSiteStrictMode,
	}
}

// setSessionCookie writes the session_token cookie with the attributes
// frozen in §4.7: HttpOnly and SameSite=Strict always on (they don't depend
// on transport), Secure only when SECURE_COOKIES is set (a Secure cookie is
// silently dropped over plain HTTP, which would break local/audit testing).
func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	cookie := baseSessionCookie()
	cookie.Value = token
	cookie.Expires = expiresAt
	http.SetCookie(w, cookie)
}

// clearSessionCookie tells the browser to delete the session_token cookie
// immediately, via the standard "MaxAge < 0" convention.
func clearSessionCookie(w http.ResponseWriter) {
	cookie := baseSessionCookie()
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}

// RegisterHandler creates a new user and immediately logs them in. Required
// fields are rejected server-side (empty/whitespace-only), and a duplicate
// email/username is turned into a friendly message instead of leaking the
// database's raw UNIQUE constraint error to the client.
func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid form submission.")
			return
		}

		email := strings.TrimSpace(r.FormValue("email"))
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		if email == "" || username == "" || strings.TrimSpace(password) == "" {
			webutil.RenderTemplate(w, "register.html", AuthPageData{
				ErrorMessage: "Email, username, and password are all required.",
			})
			return
		}

		hash, err := HashPassword(password)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not process your password.")
			return
		}

		result, err := db.Exec(
			"INSERT INTO users (username, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
			username, email, hash, nowString(),
		)
		if err != nil {
			if isUniqueConstraintError(err) {
				webutil.RenderTemplate(w, "register.html", AuthPageData{
					ErrorMessage: "That email or username is already taken.",
				})
				return
			}
			webutil.RenderError(w, http.StatusInternalServerError, "Could not create your account.")
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not create your account.")
			return
		}

		token, expiresAt, err := CreateSession(db, userID)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Account created, but sign-in failed. Please log in.")
			return
		}

		setSessionCookie(w, token, expiresAt)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// isUniqueConstraintError reports whether err came from SQLite's UNIQUE
// constraint (duplicate email or username) rather than some other failure,
// so callers can show a friendly message instead of a generic one.
func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT")
}

// genericLoginError is shown for every login failure — wrong password,
// unknown email, or an empty field all produce the same message, so the
// response never reveals whether a given email is registered at all.
const genericLoginError = "Invalid email or password."

// renderLoginError re-renders login.html with the generic login failure
// message. LoginHandler has three separate failure paths (empty fields,
// unknown email, wrong password) that must all respond identically — this
// keeps that single response in one place instead of three.
func renderLoginError(w http.ResponseWriter) {
	webutil.RenderTemplate(w, "login.html", AuthPageData{ErrorMessage: genericLoginError})
}

// LoginHandler authenticates a user by email and password and starts a new
// session on success. Every failure path returns the same generic message
// (audit: submitting the form with no credentials must show a warning, not
// a raw error or a silent no-op).
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid form submission.")
			return
		}

		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		if email == "" || strings.TrimSpace(password) == "" {
			renderLoginError(w)
			return
		}

		var user models.User
		err := db.QueryRow(
			"SELECT id, password_hash FROM users WHERE email = ?", email,
		).Scan(&user.ID, &user.PasswordHash)
		if err == sql.ErrNoRows {
			renderLoginError(w)
			return
		}
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not log you in.")
			return
		}

		if !CheckPassword(user.PasswordHash, password) {
			renderLoginError(w)
			return
		}

		token, expiresAt, err := CreateSession(db, user.ID)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not log you in.")
			return
		}

		setSessionCookie(w, token, expiresAt)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// LogoutHandler destroys the caller's session, if any, and clears the
// cookie. It is safe to call with no session at all — DestroySession is
// idempotent — so logout never errors just because the user was already
// logged out. A real database failure while destroying the session is
// logged server-side but never blocks the redirect: the user's cookie is
// cleared either way, so their browser is logged out even if the stale row
// briefly lingers in the database.
func LogoutHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if err := DestroySession(db, cookie.Value); err != nil {
				log.Printf("auth: DestroySession failed: %v", err)
			}
		}
		clearSessionCookie(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// RegisterRoutes wires the auth package's routes into mux, per §4.5.
// Method-aware patterns ("GET /path", "POST /path") make http.ServeMux
// return 405 on a method mismatch automatically, instead of every handler
// needing its own r.Method check.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /register", RegisterFormHandler)
	mux.HandleFunc("POST /register", RegisterHandler(db))
	mux.HandleFunc("GET /login", LoginFormHandler)
	mux.HandleFunc("POST /login", LoginHandler(db))
	mux.HandleFunc("POST /logout", LogoutHandler(db))
}

// RegisterFormHandler renders the empty registration form for GET /register.
func RegisterFormHandler(w http.ResponseWriter, r *http.Request) {
	webutil.RenderTemplate(w, "register.html", AuthPageData{})
}

// LoginFormHandler renders the empty login form for GET /login.
func LoginFormHandler(w http.ResponseWriter, r *http.Request) {
	webutil.RenderTemplate(w, "login.html", AuthPageData{})
}
