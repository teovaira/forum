// Package models defines the domain models and custom types used throughout
// the forum application.
//
// The package contains shared data structures, identifiers, and strongly
// typed values that represent the application's business domain. These
// models are used by the repository, service, and handler layers to
// exchange data in a consistent and type-safe manner.
package models

// ReactionTarget identifies the type of content a reaction belongs to.
//
// A reaction can target either a forum post or a comment.
type ReactionTarget string

// ReactionValue represents the value of a user's reaction.
//
// A reaction can be either a like or a dislike.
type ReactionValue int

const (
	TargetPost    ReactionTarget = "post"
	TargetComment ReactionTarget = "comment"
	Like          ReactionValue  = 1
	Dislike       ReactionValue  = -1
)
