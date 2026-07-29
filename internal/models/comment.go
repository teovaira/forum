package models

import "time"

// Comment represents a comment on a forum post.
// It contains metadata about the comment author, parent post, creation time,
// content, and likes/dislikes counts.
type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Author    string
	Body      string
	CreatedAt time.Time
	Likes     int
	Dislikes  int
}
