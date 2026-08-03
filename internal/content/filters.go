package content

import (
	"database/sql"
	"strings"

	"forum/internal/models"
)

// PostCategories retrieves categories for a batch of post IDs.
//
// It queries the database for all category associations of the provided post IDs
// in a single query to avoid N+1 queries.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - postIDs: A slice of post IDs to fetch categories for.
//
// Returns:
//   - A map associating each post ID with its corresponding category slice.
//   - An error if the database query fails.
func PostCategories(db *sql.DB, postIDs []int64) (map[int64][]models.Category, error) {
	if len(postIDs) == 0 {
		return map[int64][]models.Category{}, nil
	}

	var pcArgs []any
	var pcPlaceholders []string
	for _, id := range postIDs {
		pcPlaceholders = append(pcPlaceholders, "?")
		pcArgs = append(pcArgs, id)
	}

	pcQuery := "SELECT pc.post_id, c.id, c.name, c.kind FROM post_categories pc JOIN categories c ON pc.category_id = c.id WHERE pc.post_id IN (" + strings.Join(pcPlaceholders, ", ") + ")"

	rows, err := db.Query(pcQuery, pcArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postCatMap := make(map[int64][]models.Category)
	for rows.Next() {
		var postID int64
		var cat models.Category
		if err := rows.Scan(&postID, &cat.ID, &cat.Name, &cat.Kind); err != nil {
			return nil, err
		}
		postCatMap[postID] = append(postCatMap[postID], cat)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Ensure every ID in postIDs is represented in the map, even if it has no categories.
	for _, id := range postIDs {
		if _, ok := postCatMap[id]; !ok {
			postCatMap[id] = []models.Category{}
		}
	}

	return postCatMap, nil
}
