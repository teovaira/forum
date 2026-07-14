package content

import (
	"database/sql"
	"forum/internal/models"
)

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
