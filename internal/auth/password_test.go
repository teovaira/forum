package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash1, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}
	if hash1 == "" {
		t.Fatal("HashPassword returned an empty hash")
	}

	hash2, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}
	if hash1 == hash2 {
		t.Error("hashing the same password twice produced identical hashes; expected different salts")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	if !CheckPassword(hash, "correct horse battery staple") {
		t.Error("CheckPassword returned false for the correct password")
	}

	if CheckPassword(hash, "wrong password") {
		t.Error("CheckPassword returned true for an incorrect password")
	}
}
