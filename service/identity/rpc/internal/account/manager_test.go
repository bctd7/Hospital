package account

import (
	"context"
	"errors"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/identity/rpc/internal/login"
	"hospital/service/identity/rpc/internal/session"
)

func TestSetMyPhoneNormalizesAndMasks(t *testing.T) {
	store := &fakeStore{}
	manager, err := NewManager(store, fakeProvider{}, fakeSessions{}, "wx-app", make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := manager.SetMyPhone(context.Background(), "account-1", "138-0013-8000")
	if err != nil {
		t.Fatal(err)
	}
	if binding.Masked != "138****8000" || len(store.fingerprint) != sha256Size {
		t.Fatalf("unexpected binding: %#v fingerprint=%x", binding, store.fingerprint)
	}
}

func TestFindByPhoneRequiresAuthorizationPermission(t *testing.T) {
	manager, err := NewManager(&fakeStore{}, fakeProvider{}, fakeSessions{}, "wx-app", make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.FindByPhone(context.Background(), authn.Principal{AccountID: "patient"}, "13800138000")
	if err == nil {
		t.Fatal("expected permission denial")
	}
	_, err = manager.FindByPhone(context.Background(), authn.Principal{
		AccountID: "admin", Status: authn.AccountStatusActive,
		Permissions: []string{contractauthz.PermissionIdentityAuthorizationManage},
	}, "13800138000")
	if err != nil {
		t.Fatal(err)
	}
}

const sha256Size = 32

type fakeStore struct{ fingerprint []byte }

func (s *fakeStore) FindOrCreateWeChatAccount(context.Context, string, string) (string, error) {
	return "account-1", nil
}

func (s *fakeStore) SetSelfReportedPhone(_ context.Context, _ string, fingerprint []byte, masked string) (PhoneBinding, error) {
	s.fingerprint = append([]byte(nil), fingerprint...)
	return PhoneBinding{Masked: masked, VerificationStatus: PhoneStatusSelfReported, VerificationSource: PhoneSourceSelfReported}, nil
}

func (s *fakeStore) FindAccountByPhone(context.Context, []byte) (Lookup, error) {
	return Lookup{}, nil
}

type fakeProvider struct{ err error }

func (f fakeProvider) ExchangeLoginCode(context.Context, string) (login.WeChatSession, error) {
	return login.WeChatSession{OpenID: "openid"}, f.err
}

func (fakeProvider) ExchangePhoneCode(context.Context, string) (login.WeChatPhone, error) {
	return login.WeChatPhone{}, errors.New("not implemented")
}

type fakeSessions struct{}

func (fakeSessions) Start(context.Context, string) (session.TokenPair, error) {
	return session.TokenPair{}, nil
}
