package context

import (
	"context"
	"errors"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const (
	readerAccountID = "00000000-0000-0000-0000-000000000001"
	targetAccountID = "00000000-0000-0000-0000-000000000002"
)

func TestGetAuthorizationContextReturnsOwnPrincipalWithoutStoreRead(t *testing.T) {
	store := &authorizationReadStore{t: t}
	manager, err := NewManager(store)
	if err != nil {
		t.Fatal(err)
	}
	operator := authn.Principal{AccountID: readerAccountID, Status: authn.AccountStatusActive}

	result, err := manager.GetAuthorizationContext(context.Background(), operator, readerAccountID)
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountID != readerAccountID || store.calls != 0 {
		t.Fatalf("unexpected own authorization lookup: result=%#v calls=%d", result, store.calls)
	}
}

func TestGetAuthorizationContextRequiresPermissionForAnotherAccount(t *testing.T) {
	store := &authorizationReadStore{t: t}
	manager, err := NewManager(store)
	if err != nil {
		t.Fatal(err)
	}
	operator := authn.Principal{AccountID: readerAccountID, Status: authn.AccountStatusActive}

	_, err = manager.GetAuthorizationContext(context.Background(), operator, targetAccountID)
	if !errors.Is(err, ErrForbidden) || store.calls != 0 {
		t.Fatalf("expected forbidden lookup without store access, err=%v calls=%d", err, store.calls)
	}
}

func TestGetAuthorizationContextReadsAnotherAccountWithPermission(t *testing.T) {
	want := authn.Principal{AccountID: targetAccountID, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive}
	store := &authorizationReadStore{t: t, principal: want}
	manager, err := NewManager(store)
	if err != nil {
		t.Fatal(err)
	}
	operator := authn.Principal{
		AccountID: readerAccountID,
		Status:    authn.AccountStatusActive,
		Permissions: []string{
			contractauthz.PermissionIdentityAuthorizationManage,
		},
	}

	result, err := manager.GetAuthorizationContext(context.Background(), operator, targetAccountID)
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountID != want.AccountID || store.calls != 1 {
		t.Fatalf("unexpected managed authorization lookup: result=%#v calls=%d", result, store.calls)
	}
}

type authorizationReadStore struct {
	t         *testing.T
	principal authn.Principal
	calls     int
}

func (s *authorizationReadStore) GetAuthorizationContext(_ context.Context, accountID string) (authn.Principal, error) {
	s.calls++
	if accountID != targetAccountID {
		s.t.Fatalf("unexpected account ID %q", accountID)
	}
	if s.principal.AccountID == "" {
		return authn.Principal{}, ErrNotFound
	}
	return s.principal, nil
}
