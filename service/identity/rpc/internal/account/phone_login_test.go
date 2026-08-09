package account

import (
	"context"
	"errors"
	"testing"

	"hospital/service/identity/rpc/internal/login"
	"hospital/service/identity/rpc/internal/session"
)

func TestPhoneLoginVerifiesBeforeCreatingSession(t *testing.T) {
	provider := &fakePhoneVerificationProvider{}
	store := &fakePhoneLoginStore{accountID: "account-1"}
	sessions := &recordingSessionStarter{}
	manager, err := NewPhoneLoginManager(store, provider, sessions, make([]byte, 32), 60)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Login(context.Background(), "138-0013-8000", "123456"); err != nil {
		t.Fatal(err)
	}
	if provider.verifiedPhone != "+8613800138000" || provider.code != "123456" {
		t.Fatalf("unexpected provider input: phone=%q code=%q", provider.verifiedPhone, provider.code)
	}
	if len(store.fingerprint) != sha256Size || store.masked != "138****8000" {
		t.Fatalf("unexpected phone binding: fingerprint=%x masked=%q", store.fingerprint, store.masked)
	}
	if sessions.accountID != "account-1" {
		t.Fatalf("unexpected session account: %q", sessions.accountID)
	}
}

func TestPhoneLoginDoesNotCreateAccountWhenVerificationFails(t *testing.T) {
	provider := &fakePhoneVerificationProvider{verifyErr: login.ErrInvalidCredential}
	store := &fakePhoneLoginStore{accountID: "account-1"}
	manager, err := NewPhoneLoginManager(store, provider, &recordingSessionStarter{}, make([]byte, 32), 60)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.Login(context.Background(), "13800138000", "000000")
	if !errors.Is(err, login.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
	if store.calls != 0 {
		t.Fatalf("account store called before successful verification: %d", store.calls)
	}
}

func TestSendPhoneLoginCodeUsesNormalizedNumber(t *testing.T) {
	provider := &fakePhoneVerificationProvider{}
	manager, err := NewPhoneLoginManager(&fakePhoneLoginStore{}, provider, &recordingSessionStarter{}, make([]byte, 32), 75)
	if err != nil {
		t.Fatal(err)
	}

	retryAfter, err := manager.SendCode(context.Background(), "138 0013 8000")
	if err != nil {
		t.Fatal(err)
	}
	if provider.sentPhone != "+8613800138000" || retryAfter != 75 {
		t.Fatalf("unexpected send result: phone=%q retry=%d", provider.sentPhone, retryAfter)
	}
}

type fakePhoneVerificationProvider struct {
	sentPhone     string
	verifiedPhone string
	code          string
	verifyErr     error
}

func (p *fakePhoneVerificationProvider) SendLoginCode(_ context.Context, phone string) error {
	p.sentPhone = phone
	return nil
}

func (p *fakePhoneVerificationProvider) VerifyLoginCode(_ context.Context, phone, code string) error {
	p.verifiedPhone = phone
	p.code = code
	return p.verifyErr
}

type fakePhoneLoginStore struct {
	accountID   string
	fingerprint []byte
	masked      string
	calls       int
}

func (s *fakePhoneLoginStore) FindOrCreateVerifiedPhoneAccount(_ context.Context, fingerprint []byte, masked string) (string, error) {
	s.calls++
	s.fingerprint = append([]byte(nil), fingerprint...)
	s.masked = masked
	return s.accountID, nil
}

type recordingSessionStarter struct{ accountID string }

func (s *recordingSessionStarter) Start(_ context.Context, accountID string) (session.TokenPair, error) {
	s.accountID = accountID
	return session.TokenPair{}, nil
}
