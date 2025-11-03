package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword(DefaultArgon, "Password123!")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	ok, err := VerifyPassword("Password123!", hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatalf("expected VerifyPassword to succeed")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	ok, err := VerifyPassword("Password123!", "invalid-hash-format")
	if err == nil {
		t.Fatalf("expected error for malformed hash")
	}
	if ok {
		t.Fatalf("expected verification failure for malformed hash")
	}
}
