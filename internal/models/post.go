// Package models defines the domain models and custom types used throughout
// the forum application.
package models

import "time"

// Post represents a forum post created by a user.
// It holds metadata about the author, title, content, creation time, categories,
// and likes/dislikes counts.
type Post struct {
	ID         int64
	UserID     int64
	Author     string
	Title      string
	Body       string
	CreatedAt  time.Time
	Categories []Category
	Likes      int
	Dislikes   int
}
