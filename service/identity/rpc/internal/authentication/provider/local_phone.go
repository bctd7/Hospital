package provider

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"unicode"
)

// LocalPhoneVerificationProvider 在本地和测试环境使用固定验证码，避免调用外部短信服务。
// 生产环境禁用规则由 svc/login_providers.go 在创建本 Provider 前强制检查。
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
