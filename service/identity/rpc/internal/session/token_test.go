package session

import "testing"

func TestTokenHashMatchesGeneratedHashes(t *testing.T) {
	expected := hashRefreshToken("refresh-token")
	if !tokenHashMatches(expected, expected) {
		t.Fatal("same refresh token hash must match")
	}
	if tokenHashMatches(expected, hashRefreshToken("another-token")) {
		t.Fatal("different refresh token hashes must not match")
	}
	if tokenHashMatches("", expected) {
		t.Fatal("missing stored hash must not match a generated hash")
	}
}
