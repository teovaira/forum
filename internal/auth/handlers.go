package auth

import (
	"database/sql"
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

// setSessionCookie writes the session_token cookie with the attributes
// frozen in §4.7: HttpOnly and SameSite=Strict always on (they don't depend
// on transport), Secure only when SECURE_COOKIES is set (a Secure cookie is
// silently dropped over plain HTTP, which would break local/audit testing).
func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   os.Getenv("SECURE_COOKIES") == "true",
		SameSite: http.SameSiteStrictMode,
	})
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
			username, email, hash, time.Now().Format(time.RFC3339),
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
			webutil.RenderTemplate(w, "login.html", AuthPageData{ErrorMessage: genericLoginError})
			return
		}

		var user models.User
		err := db.QueryRow(
			"SELECT id, password_hash FROM users WHERE email = ?", email,
		).Scan(&user.ID, &user.PasswordHash)
		if err == sql.ErrNoRows {
			webutil.RenderTemplate(w, "login.html", AuthPageData{ErrorMessage: genericLoginError})
			return
		}
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not log you in.")
			return
		}

		if !CheckPassword(user.PasswordHash, password) {
			webutil.RenderTemplate(w, "login.html", AuthPageData{ErrorMessage: genericLoginError})
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
