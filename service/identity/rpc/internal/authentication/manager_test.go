package authentication

import (
	"context"
	"errors"
	"testing"

	"hospital/service/identity/rpc/internal/authentication/provider"
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

const sha256Size = 32

type fakeStore struct{ fingerprint []byte }

func (s *fakeStore) FindOrCreateWeChatAccount(context.Context, string, string) (string, error) {
	return "account-1", nil
}

func (s *fakeStore) SetSelfReportedPhone(_ context.Context, _ string, fingerprint []byte, masked string) (PhoneBinding, error) {
	s.fingerprint = append([]byte(nil), fingerprint...)
	return PhoneBinding{Masked: masked, VerificationStatus: PhoneStatusSelfReported, VerificationSource: PhoneSourceSelfReported}, nil
}

type fakeProvider struct{ err error }

func (f fakeProvider) ExchangeLoginCode(context.Context, string) (provider.WeChatSession, error) {
	return provider.WeChatSession{OpenID: "openid"}, f.err
}

func (fakeProvider) ExchangePhoneCode(context.Context, string) (provider.WeChatPhone, error) {
	return provider.WeChatPhone{}, errors.New("not implemented")
}

type fakeSessions struct{}

func (fakeSessions) Start(context.Context, string) (session.TokenPair, error) {
	return session.TokenPair{}, nil
}
