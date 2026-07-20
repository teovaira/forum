// Package models defines the domain models and custom types used throughout
// the forum application.
//
// The package contains shared data structures, identifiers, and strongly
// typed values that represent the application's business domain. These
// models are used by the repository, service, and handler layers to
// exchange data in a consistent and type-safe manner.
package models

// Category represents a forum category used to classify posts.
//
// Each category has a unique identifier, a display name, and a kind that
// groups it into one of four classification types: demographic, genre,
// theme, or discussion.
type Category struct {
	// ID is the unique identifier of the category.
	ID int64
	// Name is the display name of the category (e.g. "Shonen", "Action").
	Name string
	// Kind classifies the category into one of four groups:
	// "demographic", "genre", "theme", or "discussion".
	Kind string
}
