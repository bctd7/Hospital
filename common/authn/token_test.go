package authn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	manager := newTestTokenManager(t)

	raw, expiresAt, err := manager.Issue(Principal{
		AccountID: "account-1", AccountType: AccountTypeStaff, Status: AccountStatusActive,
		Roles: []string{RoleDepartmentDoctor, RoleDepartmentDoctor}, DepartmentID: "department-1",
		Permissions: []string{"appointment.read", "appointment.cancel"}, AuthorizationVersion: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected a future expiration")
	}

	principal, err := manager.Verify(raw)
	if err != nil {
		t.Fatal(err)
	}
	if principal.AccountID != "account-1" || principal.AuthorizationVersion != 4 {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if len(principal.Roles) != 1 {
		t.Fatalf("expected normalized roles, got %#v", principal.Roles)
	}
}

func TestTokenRejectsWrongVerificationKey(t *testing.T) {
	issuer := newTestTokenManager(t)
	wrongPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := NewTokenManager(TokenConfig{
		Issuer: "hospital-identity", Audience: "hospital-services", VerificationKey: wrongPublic, TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := issuer.Issue(Principal{AccountID: "account-1", Status: AccountStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenRejectsInactiveAccount(t *testing.T) {
	manager := newTestTokenManager(t)
	if _, _, err := manager.Issue(Principal{AccountID: "account-1", Status: AccountStatusDisabled}); !errors.Is(err, ErrInactiveAccount) {
		t.Fatalf("expected ErrInactiveAccount, got %v", err)
	}
}

func newTestTokenManager(t *testing.T) *TokenManager {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewTokenManager(TokenConfig{
		Issuer: "hospital-identity", Audience: "hospital-services",
		SigningKey: privateKey, VerificationKey: publicKey, TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
