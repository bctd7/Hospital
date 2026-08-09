package session

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"hospital/common/authn"
)

func TestManagerStartsAndRotatesRefreshSession(t *testing.T) {
	ctx := context.Background()
	store := &memorySessionStore{}
	principals := &fakePrincipalStore{principal: activeTestPrincipal(1)}
	manager, err := NewManager(store, principals, fakeAccessIssuer{}, 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	initial, err := manager.Start(ctx, principals.principal.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if initial.AccessToken != "access-v1" || initial.RefreshToken == "" {
		t.Fatalf("unexpected initial token pair: %#v", initial)
	}
	initialSessionID, initialHash, err := parseRefreshToken(initial.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if store.value.ID != initialSessionID || store.value.TokenHash != initialHash {
		t.Fatalf("refresh session was not persisted: %#v", store.value)
	}

	principals.principal = activeTestPrincipal(2)
	rotated, err := manager.Refresh(ctx, initial.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.AccessToken != "access-v2" || rotated.RefreshToken == initial.RefreshToken {
		t.Fatalf("refresh token was not rotated: %#v", rotated)
	}
	if store.value.AuthorizationVersion != 2 {
		t.Fatalf("authorization version was not refreshed: %d", store.value.AuthorizationVersion)
	}

	_, err = manager.Refresh(ctx, initial.RefreshToken)
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Fatalf("expected replay detection, got %v", err)
	}
	if store.exists {
		t.Fatal("replayed refresh token must revoke the current session")
	}
}

func TestManagerRejectsInactiveAccountDuringRefresh(t *testing.T) {
	ctx := context.Background()
	store := &memorySessionStore{}
	principals := &fakePrincipalStore{principal: activeTestPrincipal(1)}
	manager, err := NewManager(store, principals, fakeAccessIssuer{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return time.Now().UTC() }
	pair, err := manager.Start(ctx, principals.principal.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	principals.principal.Status = authn.AccountStatusDisabled

	_, err = manager.Refresh(ctx, pair.RefreshToken)
	if !errors.Is(err, authn.ErrInactiveAccount) {
		t.Fatalf("expected inactive account error, got %v", err)
	}
	if store.exists {
		t.Fatal("inactive account refresh must revoke the session")
	}
}

func TestManagerRevokesRefreshSession(t *testing.T) {
	ctx := context.Background()
	store := &memorySessionStore{}
	principals := &fakePrincipalStore{principal: activeTestPrincipal(1)}
	manager, err := NewManager(store, principals, fakeAccessIssuer{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := manager.Start(ctx, principals.principal.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Revoke(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if store.exists {
		t.Fatal("refresh session was not revoked")
	}
}

type memorySessionStore struct {
	value  Session
	exists bool
}

func (s *memorySessionStore) Create(_ context.Context, value Session) error {
	s.value, s.exists = value, true
	return nil
}

func (s *memorySessionStore) Get(_ context.Context, sessionID string) (Session, error) {
	if !s.exists || s.value.ID != sessionID {
		return Session{}, ErrSessionNotFound
	}
	return s.value, nil
}

func (s *memorySessionStore) Rotate(_ context.Context, sessionID, expectedHash, replacementHash string, version int64) error {
	if !s.exists || s.value.ID != sessionID {
		return ErrSessionNotFound
	}
	if !tokenHashMatches(s.value.TokenHash, expectedHash) {
		s.exists = false
		return ErrRefreshTokenReused
	}
	s.value.TokenHash = replacementHash
	s.value.AuthorizationVersion = version
	return nil
}

func (s *memorySessionStore) Revoke(_ context.Context, sessionID, expectedHash string) error {
	if !s.exists || s.value.ID != sessionID {
		return nil
	}
	if !tokenHashMatches(s.value.TokenHash, expectedHash) {
		return ErrInvalidRefreshToken
	}
	s.exists = false
	return nil
}

type fakePrincipalStore struct {
	principal authn.Principal
}

func (s *fakePrincipalStore) GetAuthorizationContext(context.Context, string) (authn.Principal, error) {
	return s.principal, nil
}

type fakeAccessIssuer struct{}

func (fakeAccessIssuer) Issue(principal authn.Principal) (string, time.Time, error) {
	return fmt.Sprintf("access-v%d", principal.AuthorizationVersion), time.Now().Add(15 * time.Minute), nil
}

func activeTestPrincipal(version int64) authn.Principal {
	return authn.Principal{
		AccountID: "00000000-0000-0000-0000-000000000001", AccountType: authn.AccountTypePatient,
		Status: authn.AccountStatusActive, AuthorizationVersion: version,
	}
}
