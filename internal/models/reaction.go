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
	// TargetPost identifies a reaction directed at a post.
	TargetPost ReactionTarget = "post"
	// TargetComment identifies a reaction directed at a comment.
	TargetComment ReactionTarget = "comment"
	// Like represents a positive reaction with value 1.
	Like ReactionValue = 1
	// Dislike represents a negative reaction with value -1.
	Dislike ReactionValue = -1
)
