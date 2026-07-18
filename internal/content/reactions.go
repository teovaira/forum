package content

import (
	"database/sql"
	"forum/internal/models"
	"time"
)

// ReactionCounts stores the total number of likes and dislikes for a target.
//
// The structure is used by CountReactions to return aggregated reaction counts
// for posts or comments. Each map entry associates a target ID with its
// corresponding number of likes and dislikes.
type ReactionCounts struct {
	Likes    int
	Dislikes int
}

// UpsertReaction inserts, updates, or removes a user's reaction for a target.
//
// If the user has no existing reaction for the specified target, a new reaction
// is inserted. If the existing reaction has the same value, it is removed. If
// the existing reaction has a different value, it is updated.
//
// Parameters:
// - db: The database connection used to execute the queries.
// - target: The type of target being reacted to (post or comment).
// - targetID: The identifier of the target receiving the reaction.
// - userID: The identifier of the user performing the reaction.
// - value: The reaction value to apply (like or dislike).
//
// Returns:
// - Nil if the operation completes successfully.
// - An error if a database query or update fails.
func UpsertReaction(db *sql.DB, target models.ReactionTarget, targetID int64, userID int64, value models.ReactionValue,
) error {
	var currentValue models.ReactionValue
	row := db.QueryRow("SELECT value FROM reactions WHERE user_id=? AND target_id=? AND target_type=?", userID, targetID, target)
	err := row.Scan(&currentValue)
	if err == sql.ErrNoRows {
		timeStamp := time.Now().UTC().Format(time.RFC3339)
		if _, err := db.Exec("INSERT INTO reactions (user_id, target_id, target_type, value, created_at) VALUES (?, ?, ?, ?, ?)", userID, targetID, target, value, timeStamp); err != nil {
			return err
		}
		return nil
	}
	if err == nil {
		if currentValue == value {
			if _, err := db.Exec("DELETE FROM reactions WHERE user_id=? AND target_id=? AND target_type=?", userID, targetID, target); err != nil {
				return err
			}
			return nil
		}
		if _, err := db.Exec("UPDATE reactions SET value=? WHERE user_id=? AND target_id=? AND target_type=?", value, userID, targetID, target); err != nil {
			return err
		}
		return nil
	}

	return err

}

// CountReactions returns the number of likes and dislikes for each target.
//
// The function counts reactions for every target ID in the provided slice and
// returns the results as a map keyed by target ID.
//
// Parameters:
// - db: The database connection used to execute the queries.
// - target: The type of target whose reactions are counted (post or comment).
// - ids: The target IDs whose reactions should be counted.
//
// Returns:
// - A map associating each target ID with its corresponding reaction counts.
// - An error if a database query fails.
func CountReactions(db *sql.DB, target models.ReactionTarget, ids []int64,
) (map[int64]ReactionCounts, error) {
	result := make(map[int64]ReactionCounts)
	for _, targetID := range ids {
		var counts ReactionCounts
		row := db.QueryRow("SELECT COUNT(*) FROM reactions WHERE target_id=? AND target_type=? AND value=?", targetID, target, models.Like)
		err := row.Scan(&counts.Likes)
		if err != nil {
			return result, err
		}
		row = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE target_id=? AND target_type=? AND value=?", targetID, target, models.Dislike)
		err = row.Scan(&counts.Dislikes)
		if err != nil {
			return result, err
		}
		result[targetID] = counts
	}
	return result, nil
}
