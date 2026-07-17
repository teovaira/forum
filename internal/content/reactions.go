package content

import (
	"database/sql"
	"forum/internal/models"
	"net/http"
)

// ReactionCounts holds the counts of likes and dislikes for a target.
type ReactionCounts struct {
	Likes    int
	Dislikes int
}

// UpsertReaction is a stub for Vasiliki's reaction upsertion logic.
func UpsertReaction(db *sql.DB, target models.ReactionTarget, targetID, userID int64, value models.ReactionValue) error {
	return nil
}

// CountReactions queries the database to count likes and dislikes for a list of target IDs.
func CountReactions(db *sql.DB, target models.ReactionTarget, ids []int64) (map[int64]ReactionCounts, error) {
	counts := make(map[int64]ReactionCounts)
	for _, id := range ids {
		counts[id] = ReactionCounts{Likes: 0, Dislikes: 0}
	}
	if len(ids) == 0 {
		return counts, nil
	}

	// Loop through IDs and fetch counts
	for _, id := range ids {
		var likes, dislikes int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM reactions WHERE target_id = ? AND target_type = ? AND value = 1",
			id, target,
		).Scan(&likes)
		if err != nil {
			return nil, err
		}

		err = db.QueryRow(
			"SELECT COUNT(*) FROM reactions WHERE target_id = ? AND target_type = ? AND value = -1",
			id, target,
		).Scan(&dislikes)
		if err != nil {
			return nil, err
		}

		counts[id] = ReactionCounts{Likes: likes, Dislikes: dislikes}
	}
	return counts, nil
}

// PostIDsLikedByUser retrieves the IDs of all posts liked by a user.
func PostIDsLikedByUser(db *sql.DB, userID int64) ([]int64, error) {
	rows, err := db.Query(
		"SELECT target_id FROM reactions WHERE user_id = ? AND target_type = 'post' AND value = 1",
		userID,
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
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

// ReactPostHandler is a stub for Vasiliki's post reaction HTTP handler.
func ReactPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// ReactCommentHandler is a stub for Vasiliki's comment reaction HTTP handler.
func ReactCommentHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
