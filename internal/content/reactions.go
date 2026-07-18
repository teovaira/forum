package content

import (
	"database/sql"
	"forum/internal/models"
	"time"
)

func UpsertReaction(db *sql.DB, userID int64, targetID int64, target models.ReactionTarget, value models.ReactionValue) error {
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
		} else {
			if _, err := db.Exec("UPDATE reactions SET value=? WHERE user_id=? AND target_id=? AND target_type=?", value, userID, targetID, target); err != nil {
				return err
			}
			return nil
		}
	}
	return err

}
