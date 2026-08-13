package sms

import (
	"context"
	"errors"
	"testing"
)

func TestLocalVerifier(t *testing.T) {
	verifier, err := NewLocalVerifier("246810")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.SendLoginCode(context.Background(), "+8613800138000"); err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyLoginCode(context.Background(), "+8613800138000", "246810"); err != nil {
		t.Fatalf("expected fixed local code to pass: %v", err)
	}
	if err := verifier.VerifyLoginCode(context.Background(), "+8613800138000", "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
}

func TestLocalVerifierRejectsUnsafeCode(t *testing.T) {
	for _, code := range []string{"", "123", "123456789", "abcdef"} {
		if _, err := NewLocalVerifier(code); err == nil {
			t.Fatalf("expected code %q to be rejected", code)
		}
	}
}
