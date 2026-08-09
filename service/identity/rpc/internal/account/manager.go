package account

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
	"hospital/service/identity/rpc/internal/authorization"
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

type Lookup struct {
	Principal authn.Principal
	Phone     PhoneBinding
}

type Store interface {
	FindOrCreateWeChatAccount(ctx context.Context, appID, openID string) (string, error)
	SetSelfReportedPhone(ctx context.Context, accountID string, fingerprint []byte, masked string) (PhoneBinding, error)
	FindAccountByPhone(ctx context.Context, fingerprint []byte) (Lookup, error)
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

func (m *Manager) FindByPhone(ctx context.Context, operator authn.Principal, rawPhone string) (Lookup, error) {
	if err := commonauthz.RequirePermission(operator, contractauthz.PermissionIdentityAuthorizationManage); err != nil {
		return Lookup{}, fmt.Errorf("%w: %v", authorization.ErrForbidden, err)
	}
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return Lookup{}, err
	}
	return m.store.FindAccountByPhone(ctx, m.fingerprint(normalized))
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
