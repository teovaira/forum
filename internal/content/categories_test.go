package content

import (
	"database/sql"
	"sort"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const categoryTestSchema = `
PRAGMA foreign_keys = ON;

CREATE TABLE users (
	id            INTEGER PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TEXT NOT NULL
);

CREATE TABLE categories (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	kind TEXT NOT NULL CHECK (
		kind IN ('demographic', 'genre', 'theme', 'discussion')
	)
);

CREATE TABLE posts (
	id         INTEGER PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE post_categories (
	post_id     INTEGER NOT NULL REFERENCES posts(id),
	category_id INTEGER NOT NULL REFERENCES categories(id),
	PRIMARY KEY (post_id, category_id)
);
`

func newCategoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(categoryTestSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func insertCategory(
	t *testing.T,
	db *sql.DB,
	name string,
	kind string,
) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO categories (name, kind) VALUES (?, ?)`,
		name,
		kind,
	)
	if err != nil {
		t.Fatalf("insert category %q: %v", name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get category %q ID: %v", name, err)
	}

	return id
}

func insertUser(t *testing.T, db *sql.DB, username string) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO users (
			username,
			email,
			password_hash,
			created_at
		) VALUES (?, ?, ?, ?)`,
		username,
		username+"@example.com",
		"test-password-hash",
		"2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert user %q: %v", username, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get user %q ID: %v", username, err)
	}

	return id
}

func insertPost(
	t *testing.T,
	db *sql.DB,
	userID int64,
	title string,
) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO posts (
			user_id,
			title,
			body,
			created_at
		) VALUES (?, ?, ?, ?)`,
		userID,
		title,
		"test body",
		"2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert post %q: %v", title, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get post %q ID: %v", title, err)
	}

	return id
}

func connectPostToCategory(
	t *testing.T,
	db *sql.DB,
	postID int64,
	categoryID int64,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO post_categories (post_id, category_id)
		VALUES (?, ?)`,
		postID,
		categoryID,
	)
	if err != nil {
		t.Fatalf(
			"connect post %d to category %d: %v",
			postID,
			categoryID,
			err,
		)
	}
}

func TestListCategoriesReturnsEmptySlice(t *testing.T) {
	db := newCategoryTestDB(t)

	got, err := ListCategories(db)
	if err != nil {
		t.Fatalf("ListCategories() error = %v, want nil", err)
	}

	if len(got) != 0 {
		t.Fatalf("ListCategories() returned %d categories, want 0", len(got))
	}
}

func TestListCategoriesReturnsAllCategories(t *testing.T) {
	db := newCategoryTestDB(t)

	insertCategory(t, db, "Shonen", "demographic")
	insertCategory(t, db, "Action", "genre")
	insertCategory(t, db, "School", "theme")
	insertCategory(t, db, "News", "discussion")

	got, err := ListCategories(db)
	if err != nil {
		t.Fatalf("ListCategories() error = %v, want nil", err)
	}

	if len(got) != 4 {
		t.Fatalf("ListCategories() returned %d categories, want 4", len(got))
	}

	gotByName := make(map[string]string, len(got))

	for _, category := range got {
		gotByName[category.Name] = category.Kind

		if category.ID <= 0 {
			t.Errorf(
				"category %q ID = %d, want a positive ID",
				category.Name,
				category.ID,
			)
		}
	}

	want := map[string]string{
		"Shonen": "demographic",
		"Action": "genre",
		"School": "theme",
		"News":   "discussion",
	}

	for name, kind := range want {
		gotKind, exists := gotByName[name]
		if !exists {
			t.Errorf("ListCategories() did not return category %q", name)
			continue
		}

		if gotKind != kind {
			t.Errorf(
				"category %q Kind = %q, want %q",
				name,
				gotKind,
				kind,
			)
		}
	}
}

func TestListCategoriesPopulatesFieldsCorrectly(t *testing.T) {
	db := newCategoryTestDB(t)

	wantID := insertCategory(t, db, "Isekai", "genre")

	got, err := ListCategories(db)
	if err != nil {
		t.Fatalf("ListCategories() error = %v, want nil", err)
	}

	if len(got) != 1 {
		t.Fatalf("ListCategories() returned %d categories, want 1", len(got))
	}

	category := got[0]

	if category.ID != wantID {
		t.Errorf("Category.ID = %d, want %d", category.ID, wantID)
	}

	if category.Name != "Isekai" {
		t.Errorf("Category.Name = %q, want %q", category.Name, "Isekai")
	}

	if category.Kind != "genre" {
		t.Errorf("Category.Kind = %q, want %q", category.Kind, "genre")
	}
}

func TestListCategoriesReturnsQueryError(t *testing.T) {
	db := newCategoryTestDB(t)

	if _, err := db.Exec(`DROP TABLE categories`); err != nil {
		t.Fatalf("drop categories table: %v", err)
	}

	got, err := ListCategories(db)
	if err == nil {
		t.Fatal("ListCategories() error = nil, want an error")
	}

	if len(got) != 0 {
		t.Errorf("ListCategories() returned %d categories, want 0", len(got))
	}
}

func TestPostIDsInCategoryReturnsTaggedPosts(t *testing.T) {
	db := newCategoryTestDB(t)

	userID := insertUser(t, db, "alice")
	actionID := insertCategory(t, db, "Action", "genre")

	firstPostID := insertPost(t, db, userID, "First post")
	secondPostID := insertPost(t, db, userID, "Second post")

	connectPostToCategory(t, db, firstPostID, actionID)
	connectPostToCategory(t, db, secondPostID, actionID)

	got, err := PostIDsInCategory(db, actionID)
	if err != nil {
		t.Fatalf("PostIDsInCategory() error = %v, want nil", err)
	}

	sort.Slice(got, func(i int, j int) bool {
		return got[i] < got[j]
	})

	want := []int64{firstPostID, secondPostID}

	sort.Slice(want, func(i int, j int) bool {
		return want[i] < want[j]
	})

	if len(got) != len(want) {
		t.Fatalf(
			"PostIDsInCategory() returned %d IDs, want %d",
			len(got),
			len(want),
		)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf(
				"PostIDsInCategory()[%d] = %d, want %d",
				index,
				got[index],
				want[index],
			)
		}
	}
}

func TestPostIDsInCategoryExcludesOtherCategories(t *testing.T) {
	db := newCategoryTestDB(t)

	userID := insertUser(t, db, "bob")
	actionID := insertCategory(t, db, "Action", "genre")
	romanceID := insertCategory(t, db, "Romance", "genre")

	actionPostID := insertPost(t, db, userID, "Action post")
	romancePostID := insertPost(t, db, userID, "Romance post")

	connectPostToCategory(t, db, actionPostID, actionID)
	connectPostToCategory(t, db, romancePostID, romanceID)

	got, err := PostIDsInCategory(db, actionID)
	if err != nil {
		t.Fatalf("PostIDsInCategory() error = %v, want nil", err)
	}

	if len(got) != 1 {
		t.Fatalf("PostIDsInCategory() returned %d IDs, want 1", len(got))
	}

	if got[0] != actionPostID {
		t.Errorf(
			"PostIDsInCategory()[0] = %d, want %d",
			got[0],
			actionPostID,
		)
	}
}

func TestPostIDsInCategorySupportsMultipleCategoriesPerPost(t *testing.T) {
	db := newCategoryTestDB(t)

	userID := insertUser(t, db, "carol")
	shonenID := insertCategory(t, db, "Shonen", "demographic")
	isekaiID := insertCategory(t, db, "Isekai", "genre")
	fantasyID := insertCategory(t, db, "Fantasy", "genre")

	postID := insertPost(t, db, userID, "Multi-category post")

	connectPostToCategory(t, db, postID, shonenID)
	connectPostToCategory(t, db, postID, isekaiID)
	connectPostToCategory(t, db, postID, fantasyID)

	categoryIDs := []int64{shonenID, isekaiID, fantasyID}

	for _, categoryID := range categoryIDs {
		got, err := PostIDsInCategory(db, categoryID)
		if err != nil {
			t.Fatalf(
				"PostIDsInCategory(%d) error = %v, want nil",
				categoryID,
				err,
			)
		}

		if len(got) != 1 {
			t.Errorf(
				"PostIDsInCategory(%d) returned %d IDs, want 1",
				categoryID,
				len(got),
			)
			continue
		}

		if got[0] != postID {
			t.Errorf(
				"PostIDsInCategory(%d)[0] = %d, want %d",
				categoryID,
				got[0],
				postID,
			)
		}
	}
}

func TestPostIDsInCategoryReturnsEmptyForCategoryWithoutPosts(
	t *testing.T,
) {
	db := newCategoryTestDB(t)

	categoryID := insertCategory(t, db, "Horror", "genre")

	got, err := PostIDsInCategory(db, categoryID)
	if err != nil {
		t.Fatalf("PostIDsInCategory() error = %v, want nil", err)
	}

	if len(got) != 0 {
		t.Fatalf("PostIDsInCategory() returned %d IDs, want 0", len(got))
	}
}

func TestPostIDsInCategoryReturnsEmptyForUnknownCategory(t *testing.T) {
	db := newCategoryTestDB(t)

	got, err := PostIDsInCategory(db, 999_999)
	if err != nil {
		t.Fatalf("PostIDsInCategory() error = %v, want nil", err)
	}

	if len(got) != 0 {
		t.Fatalf("PostIDsInCategory() returned %d IDs, want 0", len(got))
	}
}

func TestPostIDsInCategoryReturnsQueryError(t *testing.T) {
	db := newCategoryTestDB(t)

	if _, err := db.Exec(`DROP TABLE post_categories`); err != nil {
		t.Fatalf("drop post_categories table: %v", err)
	}

	got, err := PostIDsInCategory(db, 1)
	if err == nil {
		t.Fatal("PostIDsInCategory() error = nil, want an error")
	}

	if len(got) != 0 {
		t.Errorf("PostIDsInCategory() returned %d IDs, want 0", len(got))
	}
}
