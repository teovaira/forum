package content

import (
	"database/sql"
	"sort"
	"testing"

	"forum/internal/database"
	"forum/internal/models"
)

const reactionTestSchema = `
CREATE TABLE users (
	id INTEGER PRIMARY KEY
);

CREATE TABLE reactions (
	id          INTEGER PRIMARY KEY,
	user_id     INTEGER NOT NULL REFERENCES users(id),
	target_id   INTEGER NOT NULL,
	target_type TEXT NOT NULL CHECK (
		target_type IN ('post', 'comment')
	),
	value       INTEGER NOT NULL CHECK (
		value IN (1, -1)
	),
	created_at  TEXT NOT NULL,
	UNIQUE (user_id, target_id, target_type)
);
`

func newReactionTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("database.Connect() error = %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(reactionTestSchema); err != nil {
		t.Fatalf("create reaction test schema: %v", err)
	}

	return db
}

func insertReactionTestUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	result, err := db.Exec(`INSERT INTO users DEFAULT VALUES`)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get test user ID: %v", err)
	}

	return userID
}

func insertTestReaction(
	t *testing.T,
	db *sql.DB,
	userID int64,
	targetID int64,
	target models.ReactionTarget,
	value models.ReactionValue,
) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO reactions (
			user_id,
			target_id,
			target_type,
			value,
			created_at
		) VALUES (?, ?, ?, ?, ?)`,
		userID,
		targetID,
		target,
		value,
		"2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert test reaction: %v", err)
	}

	reactionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get reaction ID: %v", err)
	}

	return reactionID
}

func reactionRow(
	t *testing.T,
	db *sql.DB,
	userID int64,
	targetID int64,
	target models.ReactionTarget,
) (int64, models.ReactionValue, string, bool) {
	t.Helper()

	var reactionID int64
	var value models.ReactionValue
	var createdAt string

	err := db.QueryRow(
		`SELECT id, value, created_at
		FROM reactions
		WHERE user_id = ?
		  AND target_id = ?
		  AND target_type = ?`,
		userID,
		targetID,
		target,
	).Scan(&reactionID, &value, &createdAt)

	if err == sql.ErrNoRows {
		return 0, 0, "", false
	}

	if err != nil {
		t.Fatalf("query reaction row: %v", err)
	}

	return reactionID, value, createdAt, true
}

func reactionCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int

	if err := db.QueryRow(`SELECT COUNT(*) FROM reactions`).Scan(&count); err != nil {
		t.Fatalf("count reactions: %v", err)
	}

	return count
}

func TestUpsertReactionInsertsLike(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	)
	if err != nil {
		t.Fatalf("UpsertReaction() error = %v, want nil", err)
	}

	_, value, createdAt, exists := reactionRow(
		t,
		db,
		userID,
		10,
		models.TargetPost,
	)

	if !exists {
		t.Fatal("reaction does not exist after inserting a like")
	}

	if value != models.Like {
		t.Errorf("reaction value = %d, want %d", value, models.Like)
	}

	if createdAt == "" {
		t.Error("reaction created_at is empty")
	}
}

func TestUpsertReactionInsertsDislike(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Dislike,
	)
	if err != nil {
		t.Fatalf("UpsertReaction() error = %v, want nil", err)
	}

	_, value, _, exists := reactionRow(
		t,
		db,
		userID,
		10,
		models.TargetPost,
	)

	if !exists {
		t.Fatal("reaction does not exist after inserting a dislike")
	}

	if value != models.Dislike {
		t.Errorf("reaction value = %d, want %d", value, models.Dislike)
	}
}

func TestUpsertReactionRemovesSameReaction(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	); err != nil {
		t.Fatalf("first UpsertReaction() error = %v", err)
	}

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	); err != nil {
		t.Fatalf("second UpsertReaction() error = %v", err)
	}

	_, _, _, exists := reactionRow(
		t,
		db,
		userID,
		10,
		models.TargetPost,
	)

	if exists {
		t.Fatal("same reaction was not removed")
	}
}

func TestUpsertReactionChangesLikeToDislike(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	); err != nil {
		t.Fatalf("insert like: %v", err)
	}

	originalID, _, _, exists := reactionRow(
		t,
		db,
		userID,
		10,
		models.TargetPost,
	)
	if !exists {
		t.Fatal("like does not exist before update")
	}

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Dislike,
	); err != nil {
		t.Fatalf("change like to dislike: %v", err)
	}

	updatedID, value, _, exists := reactionRow(
		t,
		db,
		userID,
		10,
		models.TargetPost,
	)
	if !exists {
		t.Fatal("reaction does not exist after update")
	}

	if value != models.Dislike {
		t.Errorf("reaction value = %d, want %d", value, models.Dislike)
	}

	if updatedID != originalID {
		t.Errorf(
			"reaction ID changed from %d to %d; want update in place",
			originalID,
			updatedID,
		)
	}

	if reactionCount(t, db) != 1 {
		t.Fatalf("reaction count = %d, want 1", reactionCount(t, db))
	}
}

func TestUpsertReactionChangesDislikeToLike(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	if err := UpsertReaction(
		db,
		models.TargetComment,
		20,
		userID,
		models.Dislike,
	); err != nil {
		t.Fatalf("insert dislike: %v", err)
	}

	if err := UpsertReaction(
		db,
		models.TargetComment,
		20,
		userID,
		models.Like,
	); err != nil {
		t.Fatalf("change dislike to like: %v", err)
	}

	_, value, _, exists := reactionRow(
		t,
		db,
		userID,
		20,
		models.TargetComment,
	)

	if !exists {
		t.Fatal("reaction does not exist after update")
	}

	if value != models.Like {
		t.Errorf("reaction value = %d, want %d", value, models.Like)
	}
}

func TestUpsertReactionKeepsPostAndCommentTargetsSeparate(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	); err != nil {
		t.Fatalf("react to post: %v", err)
	}

	if err := UpsertReaction(
		db,
		models.TargetComment,
		10,
		userID,
		models.Dislike,
	); err != nil {
		t.Fatalf("react to comment: %v", err)
	}

	if reactionCount(t, db) != 2 {
		t.Fatalf("reaction count = %d, want 2", reactionCount(t, db))
	}
}

func TestUpsertReactionKeepsUsersSeparate(t *testing.T) {
	db := newReactionTestDB(t)
	firstUserID := insertReactionTestUser(t, db)
	secondUserID := insertReactionTestUser(t, db)

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		firstUserID,
		models.Like,
	); err != nil {
		t.Fatalf("first user reaction: %v", err)
	}

	if err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		secondUserID,
		models.Dislike,
	); err != nil {
		t.Fatalf("second user reaction: %v", err)
	}

	if reactionCount(t, db) != 2 {
		t.Fatalf("reaction count = %d, want 2", reactionCount(t, db))
	}
}

func TestUpsertReactionRejectsInvalidValue(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.ReactionValue(5),
	)
	if err == nil {
		t.Fatal("UpsertReaction() error = nil, want invalid-value error")
	}

	if reactionCount(t, db) != 0 {
		t.Fatalf("reaction count = %d, want 0", reactionCount(t, db))
	}
}

func TestUpsertReactionRejectsInvalidTarget(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	err := UpsertReaction(
		db,
		models.ReactionTarget("invalid"),
		10,
		userID,
		models.Like,
	)
	if err == nil {
		t.Fatal("UpsertReaction() error = nil, want invalid-target error")
	}

	if reactionCount(t, db) != 0 {
		t.Fatalf("reaction count = %d, want 0", reactionCount(t, db))
	}
}

func TestUpsertReactionReturnsDatabaseError(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	err := UpsertReaction(
		db,
		models.TargetPost,
		10,
		userID,
		models.Like,
	)
	if err == nil {
		t.Fatal("UpsertReaction() error = nil, want database error")
	}
}

func TestCountReactionsCountsLikesAndDislikesForPosts(t *testing.T) {
	db := newReactionTestDB(t)

	firstUserID := insertReactionTestUser(t, db)
	secondUserID := insertReactionTestUser(t, db)
	thirdUserID := insertReactionTestUser(t, db)

	insertTestReaction(
		t,
		db,
		firstUserID,
		10,
		models.TargetPost,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		secondUserID,
		10,
		models.TargetPost,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		thirdUserID,
		10,
		models.TargetPost,
		models.Dislike,
	)
	insertTestReaction(
		t,
		db,
		firstUserID,
		20,
		models.TargetPost,
		models.Dislike,
	)

	got, err := CountReactions(
		db,
		models.TargetPost,
		[]int64{10, 20},
	)
	if err != nil {
		t.Fatalf("CountReactions() error = %v, want nil", err)
	}

	if got[10].Likes != 2 {
		t.Errorf("post 10 likes = %d, want 2", got[10].Likes)
	}

	if got[10].Dislikes != 1 {
		t.Errorf("post 10 dislikes = %d, want 1", got[10].Dislikes)
	}

	if got[20].Likes != 0 {
		t.Errorf("post 20 likes = %d, want 0", got[20].Likes)
	}

	if got[20].Dislikes != 1 {
		t.Errorf("post 20 dislikes = %d, want 1", got[20].Dislikes)
	}
}

func TestCountReactionsCountsCommentsSeparately(t *testing.T) {
	db := newReactionTestDB(t)

	firstUserID := insertReactionTestUser(t, db)
	secondUserID := insertReactionTestUser(t, db)

	insertTestReaction(
		t,
		db,
		firstUserID,
		10,
		models.TargetPost,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		secondUserID,
		10,
		models.TargetComment,
		models.Dislike,
	)

	got, err := CountReactions(
		db,
		models.TargetComment,
		[]int64{10},
	)
	if err != nil {
		t.Fatalf("CountReactions() error = %v, want nil", err)
	}

	if got[10].Likes != 0 {
		t.Errorf("comment 10 likes = %d, want 0", got[10].Likes)
	}

	if got[10].Dislikes != 1 {
		t.Errorf("comment 10 dislikes = %d, want 1", got[10].Dislikes)
	}
}

func TestCountReactionsReturnsZeroCountsForTargetsWithoutReactions(
	t *testing.T,
) {
	db := newReactionTestDB(t)

	got, err := CountReactions(
		db,
		models.TargetPost,
		[]int64{10},
	)
	if err != nil {
		t.Fatalf("CountReactions() error = %v, want nil", err)
	}

	if got[10].Likes != 0 {
		t.Errorf("post 10 likes = %d, want 0", got[10].Likes)
	}

	if got[10].Dislikes != 0 {
		t.Errorf("post 10 dislikes = %d, want 0", got[10].Dislikes)
	}
}

func TestCountReactionsReturnsEmptyMapForEmptyIDs(t *testing.T) {
	db := newReactionTestDB(t)

	got, err := CountReactions(
		db,
		models.TargetPost,
		nil,
	)
	if err != nil {
		t.Fatalf("CountReactions() error = %v, want nil", err)
	}

	if len(got) != 0 {
		t.Fatalf("CountReactions() returned %d entries, want 0", len(got))
	}
}

func TestCountReactionsReturnsQueryError(t *testing.T) {
	db := newReactionTestDB(t)

	if _, err := db.Exec(`DROP TABLE reactions`); err != nil {
		t.Fatalf("drop reactions table: %v", err)
	}

	got, err := CountReactions(
		db,
		models.TargetPost,
		[]int64{10},
	)
	if err == nil {
		t.Fatal("CountReactions() error = nil, want an error")
	}

	if len(got) != 0 {
		t.Errorf("CountReactions() returned %d entries, want 0", len(got))
	}
}

func TestPostIDsLikedByUserReturnsLikedPostIDs(t *testing.T) {
	db := newReactionTestDB(t)

	userID := insertReactionTestUser(t, db)
	otherUserID := insertReactionTestUser(t, db)

	insertTestReaction(
		t,
		db,
		userID,
		10,
		models.TargetPost,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		userID,
		20,
		models.TargetPost,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		userID,
		30,
		models.TargetPost,
		models.Dislike,
	)
	insertTestReaction(
		t,
		db,
		userID,
		40,
		models.TargetComment,
		models.Like,
	)
	insertTestReaction(
		t,
		db,
		otherUserID,
		50,
		models.TargetPost,
		models.Like,
	)

	got, err := PostIDsLikedByUser(db, userID)
	if err != nil {
		t.Fatalf("PostIDsLikedByUser() error = %v, want nil", err)
	}

	sort.Slice(got, func(i int, j int) bool {
		return got[i] < got[j]
	})

	want := []int64{10, 20}

	if len(got) != len(want) {
		t.Fatalf(
			"PostIDsLikedByUser() returned %d IDs, want %d",
			len(got),
			len(want),
		)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf(
				"PostIDsLikedByUser()[%d] = %d, want %d",
				index,
				got[index],
				want[index],
			)
		}
	}
}

func TestPostIDsLikedByUserReturnsEmptyWhenUserHasNoLikes(t *testing.T) {
	db := newReactionTestDB(t)
	userID := insertReactionTestUser(t, db)

	got, err := PostIDsLikedByUser(db, userID)
	if err != nil {
		t.Fatalf("PostIDsLikedByUser() error = %v, want nil", err)
	}

	if len(got) != 0 {
		t.Fatalf("PostIDsLikedByUser() returned %d IDs, want 0", len(got))
	}
}

func TestPostIDsLikedByUserReturnsQueryError(t *testing.T) {
	db := newReactionTestDB(t)

	if _, err := db.Exec(`DROP TABLE reactions`); err != nil {
		t.Fatalf("drop reactions table: %v", err)
	}

	got, err := PostIDsLikedByUser(db, 1)
	if err == nil {
		t.Fatal("PostIDsLikedByUser() error = nil, want an error")
	}

	if len(got) != 0 {
		t.Errorf("PostIDsLikedByUser() returned %d IDs, want 0", len(got))
	}
}
