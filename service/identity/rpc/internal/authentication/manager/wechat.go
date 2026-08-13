// Package manager 负责验证登录凭据、解析对应账号并开始会话。
// 外部渠道调用属于 provider，账号管理属于 account/manager，会话生命周期属于 session。
package manager

import (
	"context"
	"errors"
	"strings"

	"hospital/service/identity/rpc/internal/authentication/provider"
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
	provider    provider.WeChatProvider
	sessions    SessionStarter
	wechatAppID string
	phoneKey    []byte
}

func NewManager(store Store, loginProvider provider.WeChatProvider, sessions SessionStarter, wechatAppID string, phoneKey []byte) (*Manager, error) {
	if store == nil || loginProvider == nil || sessions == nil {
		return nil, errors.New("identity account dependencies are required")
	}
	if len(phoneKey) < 32 {
		return nil, errors.New("identity phone lookup key must contain at least 32 bytes")
	}
	return &Manager{
		store: store, provider: loginProvider, sessions: sessions,
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
