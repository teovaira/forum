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
)

// ListCategories retrieves all forum categories from the database.
//
// The function queries the categories table and returns every stored category,
// including its ID, name, and kind.
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - A slice containing every category stored in the database.
//   - An error if the query, row scanning, or iteration fails.
func ListCategories(db *sql.DB) ([]models.Category, error) {
	var categories []models.Category
	rows, err := db.Query("SELECT id, name, kind FROM categories")
	if err != nil {
		return categories, err
	}
	defer rows.Close()
	for rows.Next() {
		var category models.Category
		err = rows.Scan(&category.ID, &category.Name, &category.Kind)

		if err != nil {
			return categories, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return categories, err
	}
	return categories, nil
}

// PostIDsInCategory retrieves the IDs of all posts assigned to a category.
//
// The function queries the post_categories table and returns the IDs of every
// post associated with the specified category.
//
// Parameters:
//   - db: An open SQLite database connection.
//   - categoryID: The ID of the category whose posts should be retrieved.
//
// Returns:
//   - A slice containing the IDs of all posts assigned to the category.
//   - An error if the query, row scanning, or iteration fails.
func PostIDsInCategory(db *sql.DB, categoryID int64) ([]int64, error) {
	var postIDs []int64
	rows, err := db.Query("SELECT post_id FROM post_categories WHERE category_id=?", categoryID)
	if err != nil {
		return postIDs, err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		err = rows.Scan(&postID)
		if err != nil {
			return postIDs, err
		}
		postIDs = append(postIDs, postID)
	}
	if err = rows.Err(); err != nil {
		return postIDs, err

	}
	return postIDs, nil
}
