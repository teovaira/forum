// Package main is the forum server's entry point. It wires together
// internal/database, internal/auth, and internal/content, then starts
// listening for HTTP requests.
package main

import "os"

// lookupEnv returns the value of the environment variable named key, falling
// back to fallback when that variable is unset or empty. An empty value is
// treated as unset because an exported but blank PORT would otherwise build
// the listen address ":", binding an arbitrary free port instead of 8080.
//
// Parameters:
//   - key: the name of the environment variable to read.
//   - fallback: the value to use when key is unset or empty.
//
// Returns:
//   - string: the environment value when it is set and non-empty, otherwise
//     fallback.
func lookupEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
