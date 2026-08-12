package account

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"hospital/service/identity/rpc/internal/login"
	"hospital/service/identity/rpc/internal/session"
)

const (
	PhoneStatusSelfReported = "self_reported"
	PhoneStatusVerified     = "verified"
	PhoneSourceSelfReported = "self_reported"
)

var (
	ErrInvalidPhone            = errors.New("invalid phone number")
	ErrPhoneInUse              = errors.New("phone number is already registered")
	ErrVerifiedPhoneChange     = errors.New("verified login phone must be changed through verification")
	ErrInvalidExternalIdentity = errors.New("invalid external identity")
	mainlandPhone              = regexp.MustCompile(`^1[3-9][0-9]{9}$`)
)

type PhoneBinding struct {
	Masked             string
	VerificationStatus string
	VerificationSource string
}

type Store interface {
	FindOrCreateWeChatAccount(ctx context.Context, appID, openID string) (string, error)
	SetSelfReportedPhone(ctx context.Context, accountID string, fingerprint []byte, masked string) (PhoneBinding, error)
}

type SessionStarter interface {
	Start(ctx context.Context, accountID string) (session.TokenPair, error)
}

type Manager struct {
	store       Store
	provider    login.WeChatProvider
	sessions    SessionStarter
	wechatAppID string
	phoneKey    []byte
}

func NewManager(store Store, provider login.WeChatProvider, sessions SessionStarter, wechatAppID string, phoneKey []byte) (*Manager, error) {
	if store == nil || provider == nil || sessions == nil {
		return nil, errors.New("identity account dependencies are required")
	}
	if len(phoneKey) < 32 {
		return nil, errors.New("identity phone lookup key must contain at least 32 bytes")
	}
	return &Manager{
		store: store, provider: provider, sessions: sessions,
		wechatAppID: strings.TrimSpace(wechatAppID), phoneKey: append([]byte(nil), phoneKey...),
	}, nil
}

func (m *Manager) WeChatLogin(ctx context.Context, loginCode string) (session.TokenPair, error) {
	wechatSession, err := m.provider.ExchangeLoginCode(ctx, strings.TrimSpace(loginCode))
	if err != nil {
		return session.TokenPair{}, err
	}
	accountID, err := m.store.FindOrCreateWeChatAccount(ctx, m.wechatAppID, wechatSession.OpenID)
	if err != nil {
		return session.TokenPair{}, err
	}
	return m.sessions.Start(ctx, accountID)
}

func (m *Manager) SetMyPhone(ctx context.Context, accountID, rawPhone string) (PhoneBinding, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return PhoneBinding{}, err
	}
	return m.store.SetSelfReportedPhone(ctx, accountID, m.fingerprint(normalized), maskPhone(normalized))
}

func (m *Manager) fingerprint(phone string) []byte {
	return phoneFingerprint(m.phoneKey, phone)
}

func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	value = strings.TrimPrefix(value, "+86")
	if !mainlandPhone.MatchString(value) {
		return "", ErrInvalidPhone
	}
	return "+86" + value, nil
}

func maskPhone(normalized string) string {
	digits := strings.TrimPrefix(normalized, "+86")
	return digits[:3] + "****" + digits[7:]
}
