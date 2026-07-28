package main

import "testing"

func TestLookupEnv(t *testing.T) {
	t.Run("returns the fallback when the variable is unset", func(t *testing.T) {
		if got := lookupEnv("FORUM_TEST_UNSET", "8080"); got != "8080" {
			t.Errorf("lookupEnv() = %q, want %q", got, "8080")
		}
	})

	t.Run("returns the environment value when it is set", func(t *testing.T) {
		t.Setenv("FORUM_TEST_PORT", "9999")

		if got := lookupEnv("FORUM_TEST_PORT", "8080"); got != "9999" {
			t.Errorf("lookupEnv() = %q, want %q", got, "9999")
		}
	})

	// An exported but blank PORT would otherwise build the listen address
	// ":" and bind an arbitrary free port instead of the intended one.
	t.Run("returns the fallback when the variable is set but empty", func(t *testing.T) {
		t.Setenv("FORUM_TEST_EMPTY", "")

		if got := lookupEnv("FORUM_TEST_EMPTY", "8080"); got != "8080" {
			t.Errorf("lookupEnv() = %q, want %q", got, "8080")
		}
	})
}
