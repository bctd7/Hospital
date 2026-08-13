package authentication

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"

	"hospital/service/identity/rpc/internal/authentication/provider"
	"hospital/service/identity/rpc/internal/session"
)

const PhoneSourceSMS = "sms"

var verificationCode = regexp.MustCompile(`^[0-9A-Za-z]{4,8}$`)

type PhoneLoginStore interface {
	FindOrCreateVerifiedPhoneAccount(ctx context.Context, fingerprint []byte, masked string) (string, error)
}

type PhoneLoginManager struct {
	store        PhoneLoginStore
	provider     provider.PhoneVerificationProvider
	sessions     SessionStarter
	phoneKey     []byte
	retrySeconds int64
}

func NewPhoneLoginManager(store PhoneLoginStore, phoneProvider provider.PhoneVerificationProvider, sessions SessionStarter, phoneKey []byte, retrySeconds int64) (*PhoneLoginManager, error) {
	if store == nil || phoneProvider == nil || sessions == nil {
		return nil, errors.New("phone login dependencies are required")
	}
	if len(phoneKey) < 32 {
		return nil, errors.New("identity phone lookup key must contain at least 32 bytes")
	}
	if retrySeconds <= 0 {
		retrySeconds = 60
	}
	return &PhoneLoginManager{
		store: store, provider: phoneProvider, sessions: sessions,
		phoneKey: append([]byte(nil), phoneKey...), retrySeconds: retrySeconds,
	}, nil
}

func (m *PhoneLoginManager) SendCode(ctx context.Context, rawPhone string) (int64, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return 0, err
	}
	if err := m.provider.SendLoginCode(ctx, normalized); err != nil {
		return 0, err
	}
	return m.retrySeconds, nil
}

func (m *PhoneLoginManager) Login(ctx context.Context, rawPhone, rawCode string) (session.TokenPair, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return session.TokenPair{}, err
	}
	code := strings.TrimSpace(rawCode)
	if !verificationCode.MatchString(code) {
		return session.TokenPair{}, provider.ErrInvalidCredential
	}
	if err := m.provider.VerifyLoginCode(ctx, normalized, code); err != nil {
		return session.TokenPair{}, err
	}
	accountID, err := m.store.FindOrCreateVerifiedPhoneAccount(ctx, phoneFingerprint(m.phoneKey, normalized), maskPhone(normalized))
	if err != nil {
		return session.TokenPair{}, err
	}
	return m.sessions.Start(ctx, accountID)
}

func phoneFingerprint(key []byte, phone string) []byte {
	digest := hmac.New(sha256.New, key)
	_, _ = digest.Write([]byte(phone))
	return digest.Sum(nil)
}
