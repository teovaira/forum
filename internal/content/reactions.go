// Package content provides data access functions for forum content.
//
// The package contains repository functions that retrieve and manipulate
// forum data stored in the SQLite database, including categories, posts,
// comments, and reactions. Each function encapsulates the required SQL
// queries and returns Go data structures for use by the rest of the
// application.
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
	// Likes is the total number of positive reactions for the target.
	Likes int
	// Dislikes is the total number of negative reactions for the target.
	Dislikes int
}

// UpsertReaction inserts, updates, or removes a user's reaction for a target.
//
// If the user has no existing reaction for the specified target, a new reaction
// is inserted. If the existing reaction has the same value, it is removed. If
// the existing reaction has a different value, it is updated.
//
// Parameters:
//   - db: The database connection used to execute the queries.
//   - target: The type of target being reacted to (post or comment).
//   - targetID: The identifier of the target receiving the reaction.
//   - userID: The identifier of the user performing the reaction.
//   - value: The reaction value to apply (like or dislike).
//
// Returns:
//   - Nil if the operation completes successfully.
//   - An error if a database query or update fails.
func UpsertReaction(db *sql.DB, target models.ReactionTarget, targetID int64, userID int64, value models.ReactionValue) error {
	var currentValue models.ReactionValue
	row := db.QueryRow("SELECT value FROM reactions WHERE user_id=? AND target_id=? AND target_type=?", userID, targetID, target)
	err := row.Scan(&currentValue)

	if err == sql.ErrNoRows {
		return insertReaction(db, target, targetID, userID, value)
	}
	if err != nil {
		return err
	}
	if currentValue == value {
		return deleteReaction(db, target, targetID, userID)
	}
	return updateReaction(db, target, targetID, userID, value)
}

// insertReaction adds a new reaction row for a user and target.
func insertReaction(db *sql.DB, target models.ReactionTarget, targetID, userID int64, value models.ReactionValue) error {
	timeStamp := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		"INSERT INTO reactions (user_id, target_id, target_type, value, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, targetID, target, value, timeStamp,
	)
	return err
}

// deleteReaction removes an existing reaction row for a user and target.
func deleteReaction(db *sql.DB, target models.ReactionTarget, targetID, userID int64) error {
	_, err := db.Exec(
		"DELETE FROM reactions WHERE user_id=? AND target_id=? AND target_type=?",
		userID, targetID, target,
	)
	return err
}

// updateReaction changes the value of an existing reaction row.
func updateReaction(db *sql.DB, target models.ReactionTarget, targetID, userID int64, value models.ReactionValue) error {
	_, err := db.Exec(
		"UPDATE reactions SET value=? WHERE user_id=? AND target_id=? AND target_type=?",
		value, userID, targetID, target,
	)
	return err
}

// CountReactions returns the number of likes and dislikes for each target.
//
// The function counts reactions for every target ID in the provided slice and
// returns the results as a map keyed by target ID.
//
// Parameters:
//   - db: The database connection used to execute the queries.
//   - target: The type of target whose reactions are counted (post or comment).
//   - ids: The target IDs whose reactions should be counted.
//
// Returns:
//   - A map associating each target ID with its corresponding reaction counts.
//   - An error if a database query fails.
func CountReactions(db *sql.DB, target models.ReactionTarget, ids []int64) (map[int64]ReactionCounts, error) {
	result := make(map[int64]ReactionCounts)
	for _, targetID := range ids {
		var counts ReactionCounts
		row := db.QueryRow("SELECT COUNT(*) FROM reactions WHERE target_id=? AND target_type=? AND value=?", targetID, target, models.Like)
		if err := row.Scan(&counts.Likes); err != nil {
			return result, err
		}
		row = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE target_id=? AND target_type=? AND value=?", targetID, target, models.Dislike)
		if err := row.Scan(&counts.Dislikes); err != nil {
			return result, err
		}
		result[targetID] = counts
	}
	return result, nil
}

// PostIDsLikedByUser returns the IDs of all posts liked by a user.
//
// The function retrieves every post that has a positive reaction from the
// specified user. Only reactions targeting posts with a like value are
// included in the returned slice.
//
// Parameters:
//   - db: The database connection used to execute the query.
//   - userID: The ID of the user whose liked posts are retrieved.
//
// Returns:
//   - A slice containing the IDs of all posts liked by the user.
//   - An error if the database query or row iteration fails.
func PostIDsLikedByUser(db *sql.DB, userID int64) ([]int64, error) {
	var postIDs []int64
	rows, err := db.Query(
		"SELECT target_id FROM reactions WHERE user_id=? AND target_type=? AND value=?",
		userID, models.TargetPost, models.Like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		if err = rows.Scan(&postID); err != nil {
			return nil, err
		}
		postIDs = append(postIDs, postID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(postIDs) == 0 {
		return []int64{}, nil
	}
	return postIDs, nil
}
