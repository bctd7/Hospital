package svc

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestDecodePublicKey(t *testing.T) {
	key := make([]byte, ed25519.PublicKeySize)
	encoded := base64.StdEncoding.EncodeToString(key)
	decoded, err := decodePublicKey(encoded)
	if err != nil {
		t.Fatalf("decode valid public key: %v", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		t.Fatalf("unexpected public key length: %d", len(decoded))
	}
	if _, err := decodePublicKey(""); err == nil {
		t.Fatal("expected empty public key to be rejected")
	}
}
