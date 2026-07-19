// Package content provides handlers and database operations for posts,
// comments, and reactions.
//
// This package manages the core forum functionality including creating
// and listing posts and comments, and handling user reactions.
// All handlers require a valid database connection and use the shared
// webutil render functions for responses.
package content

import (
	"database/sql"
	"fmt"
	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/webutil"
	"net/http"
	"strconv"
	"strings"
)

// ReactPostHandler handles like and dislike reactions on posts.
//
// It reads the current user from context, parses the post ID from the URL,
// determines the reaction type from the URL suffix ("/like" or "/dislike"),
// and calls UpsertReaction to apply the toggle logic. On success it
// redirects the user back to the post page.
//
// Parameters:
// - db: Active SQLite database connection.
//
// Returns:
// - An http.HandlerFunc that processes the reaction request.
func ReactPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, exists := auth.UserFromContext(r.Context())
		if !exists {
			webutil.RenderError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID := user.ID
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "invalid post id")
			return
		}
		url := r.URL.Path
		var value models.ReactionValue
		if end := strings.HasSuffix(url, "like"); end {
			value = models.Like
		} else {
			value = models.Dislike

		}
		if err = UpsertReaction(db, models.TargetPost, id, userID, value); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "failed to react")
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/posts/%d", id), http.StatusSeeOther)
	}

}
