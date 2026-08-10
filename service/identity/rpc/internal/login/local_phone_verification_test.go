package login

import (
	"context"
	"errors"
	"testing"
)

func TestLocalPhoneVerificationProvider(t *testing.T) {
	provider, err := NewLocalPhoneVerificationProvider("246810")
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.SendLoginCode(context.Background(), "+8613800138000"); err != nil {
		t.Fatal(err)
	}
	if err := provider.VerifyLoginCode(context.Background(), "+8613800138000", "246810"); err != nil {
		t.Fatalf("expected fixed local code to pass: %v", err)
	}
	if err := provider.VerifyLoginCode(context.Background(), "+8613800138000", "000000"); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("expected ErrInvalidCredential, got %v", err)
	}
}

func TestLocalPhoneVerificationProviderRejectsUnsafeCode(t *testing.T) {
	for _, code := range []string{"", "123", "1234567890123", "abcdef"} {
		if _, err := NewLocalPhoneVerificationProvider(code); err == nil {
			t.Fatalf("expected code %q to be rejected", code)
		}
	}
}
