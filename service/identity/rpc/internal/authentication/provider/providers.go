// Package provider contains adapters for external login credential providers.
// Providers verify credentials only; they never create accounts or issue tokens.
package provider

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrProviderNotConfigured = errors.New("login provider is not configured")
	ErrProviderUnavailable   = errors.New("login provider is unavailable")
	ErrInvalidCredential     = errors.New("login credential is invalid or expired")
	ErrRateLimited           = errors.New("login verification request is rate limited")
)

// PhoneVerificationProvider delegates SMS code generation and verification to
// the provider. Hospital never needs to receive or persist the plaintext code.
type PhoneVerificationProvider interface {
	SendLoginCode(ctx context.Context, phoneNumber string) error
	VerifyLoginCode(ctx context.Context, phoneNumber, code string) error
}

type UnconfiguredPhoneVerificationProvider struct{}

func (UnconfiguredPhoneVerificationProvider) SendLoginCode(context.Context, string) error {
	return ErrProviderNotConfigured
}

func (UnconfiguredPhoneVerificationProvider) VerifyLoginCode(context.Context, string, string) error {
	return ErrProviderNotConfigured
}

// LocalPhoneVerificationProvider avoids external SMS calls while retaining a
// real credential check. Environment restrictions are enforced by Identity
// service startup before this provider is constructed.
type LocalPhoneVerificationProvider struct {
	code string
}

func NewLocalPhoneVerificationProvider(code string) (*LocalPhoneVerificationProvider, error) {
	code = strings.TrimSpace(code)
	if len(code) < 4 || len(code) > 12 || strings.IndexFunc(code, func(value rune) bool {
		return !unicode.IsDigit(value)
	}) >= 0 {
		return nil, fmt.Errorf("local SMS code must contain 4 to 12 digits")
	}
	return &LocalPhoneVerificationProvider{code: code}, nil
}

func (*LocalPhoneVerificationProvider) SendLoginCode(context.Context, string) error {
	return nil
}

func (p *LocalPhoneVerificationProvider) VerifyLoginCode(_ context.Context, _ string, code string) error {
	code = strings.TrimSpace(code)
	if subtle.ConstantTimeCompare([]byte(code), []byte(p.code)) != 1 {
		return ErrInvalidCredential
	}
	return nil
}

// WeChatSession is the server-side identity returned after exchanging the
// short-lived code produced by wx.login. SessionKey must never be sent to the client.
type WeChatSession struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// WeChatPhone is returned after exchanging the one-time code produced by the
// getPhoneNumber button capability. The code is different from a wx.login code.
type WeChatPhone struct {
	PhoneNumber     string
	PurePhoneNumber string
	CountryCode     string
}

// WeChatProvider isolates Identity Service from the concrete WeChat HTTP API.
// A production adapter will keep AppSecret and access tokens only on the server.
type WeChatProvider interface {
	ExchangeLoginCode(ctx context.Context, code string) (WeChatSession, error)
	ExchangePhoneCode(ctx context.Context, code string) (WeChatPhone, error)
}

// SMSProvider only delivers a code. Identity Service remains responsible for
// generating challenges, expiration, attempt limits, rate limits and verification.
type SMSProvider interface {
	SendVerificationCode(ctx context.Context, phoneNumber, code, purpose string) error
}

// VerificationCodeStore keeps short-lived hashed challenges. Implementations
// should use Redis and must not persist plaintext verification codes.
type VerificationCodeStore interface {
	Create(ctx context.Context, challenge VerificationChallenge) error
	Consume(ctx context.Context, phoneNumber, purpose, code string) error
}

type VerificationChallenge struct {
	PhoneNumber      string
	Purpose          string
	CodeHash         string
	ExpiresInSeconds int64
	MaxAttempts      int
}
