package models

// Category represents a forum category with its identifier, name, and classification.
type Category struct {
	ID   int64
	Name string
	Kind string
}
