package content

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/webutil"
)

// ReactionCounts exists to group the aggregate like and dislike counts for a single post or comment target.
type ReactionCounts struct {
	Likes    int
	Dislikes int
}

// UpsertReaction exists to allow record upserts and deletes for a user's reaction on a post or comment.
func UpsertReaction(db *sql.DB, target models.ReactionTarget, targetID, userID int64, value models.ReactionValue) error {
	if target != models.TargetPost && target != models.TargetComment {
		return fmt.Errorf("invalid reaction target: %s", target)
	}
	if value != models.Like && value != models.Dislike {
		return fmt.Errorf("invalid reaction value: %d", value)
	}

	var existingID int64
	var existingValue int
	err := db.QueryRow(
		"SELECT id, value FROM reactions WHERE user_id = ? AND target_id = ? AND target_type = ?",
		userID, targetID, target,
	).Scan(&existingID, &existingValue)

	if err == sql.ErrNoRows {
		_, err = db.Exec(
			"INSERT INTO reactions (user_id, target_id, target_type, value, created_at) VALUES (?, ?, ?, ?, ?)",
			userID, targetID, target, value, time.Now().UTC().Format(time.RFC3339),
		)
		return err
	} else if err != nil {
		return err
	}

	if models.ReactionValue(existingValue) == value {
		_, err = db.Exec("DELETE FROM reactions WHERE id = ?", existingID)
		return err
	}

	_, err = db.Exec("UPDATE reactions SET value = ? WHERE id = ?", value, existingID)
	return err
}

// CountReactions exists to batch retrieve the counts of likes and dislikes for a set of posts or comments to avoid N+1 query patterns.
func CountReactions(db *sql.DB, target models.ReactionTarget, ids []int64) (map[int64]ReactionCounts, error) {
	result := make(map[int64]ReactionCounts)
	for _, id := range ids {
		result[id] = ReactionCounts{}
	}
	if len(ids) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = target
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := "SELECT target_id, value, COUNT(*) FROM reactions WHERE target_type = ? AND target_id IN (" +
		strings.Join(placeholders, ",") + ") GROUP BY target_id, value"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var targetID int64
		var val int
		var count int
		if err := rows.Scan(&targetID, &val, &count); err != nil {
			return nil, err
		}
		counts := result[targetID]
		if val == 1 {
			counts.Likes = count
		} else if val == -1 {
			counts.Dislikes = count
		}
		result[targetID] = counts
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// PostIDsLikedByUser exists to query the IDs of all posts liked by a specific user for filtering.
func PostIDsLikedByUser(db *sql.DB, userID int64) ([]int64, error) {
	rows, err := db.Query(
		"SELECT target_id FROM reactions WHERE user_id = ? AND target_type = ? AND value = 1",
		userID, models.TargetPost,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// ReactPostHandler exists to return an HTTP handler that updates a user's reaction to a post.
func ReactPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			webutil.RenderError(w, http.StatusUnauthorized, "You must be logged in to react.")
			return
		}

		idStr := r.PathValue("id")
		postID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid post ID.")
			return
		}

		var value models.ReactionValue
		if strings.HasSuffix(r.URL.Path, "/like") {
			value = models.Like
		} else if strings.HasSuffix(r.URL.Path, "/dislike") {
			value = models.Dislike
		} else {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid reaction type.")
			return
		}

		if err := UpsertReaction(db, models.TargetPost, postID, user.ID, value); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not process reaction.")
			return
		}

		referer := r.Referer()
		if referer == "" {
			referer = "/"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)
	}
}

// ReactCommentHandler exists to return an HTTP handler that updates a user's reaction to a comment.
func ReactCommentHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			webutil.RenderError(w, http.StatusUnauthorized, "You must be logged in to react.")
			return
		}

		idStr := r.PathValue("id")
		commentID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid comment ID.")
			return
		}

		var value models.ReactionValue
		if strings.HasSuffix(r.URL.Path, "/like") {
			value = models.Like
		} else if strings.HasSuffix(r.URL.Path, "/dislike") {
			value = models.Dislike
		} else {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid reaction type.")
			return
		}

		if err := UpsertReaction(db, models.TargetComment, commentID, user.ID, value); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not process reaction.")
			return
		}

		referer := r.Referer()
		if referer == "" {
			referer = "/"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)
	}
}
