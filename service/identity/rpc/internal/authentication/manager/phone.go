package manager

import (
	"context"
	"regexp"
	"strings"

	"hospital/service/identity/rpc/internal/authentication/sms"
	"hospital/service/identity/rpc/internal/session"
)

var verificationCode = regexp.MustCompile(`^[0-9]{4,8}$`)

func (m *Manager) SendCode(ctx context.Context, rawPhone string) (int64, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return 0, err
	}
	if err := m.verifier.SendLoginCode(ctx, normalized); err != nil {
		return 0, err
	}
	return m.retrySeconds, nil
}

func (m *Manager) Login(ctx context.Context, rawPhone, rawCode string) (session.TokenPair, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return session.TokenPair{}, err
	}
	code := strings.TrimSpace(rawCode)
	if !verificationCode.MatchString(code) {
		return session.TokenPair{}, sms.ErrInvalidCode
	}
	if err := m.verifier.VerifyLoginCode(ctx, normalized, code); err != nil {
		return session.TokenPair{}, err
	}
	accountID, err := m.store.FindOrCreateVerifiedPhoneAccount(ctx, phoneFingerprint(m.phoneKey, normalized), maskPhone(normalized))
	if err != nil {
		return session.TokenPair{}, err
	}
	return m.sessions.Start(ctx, accountID)
}
