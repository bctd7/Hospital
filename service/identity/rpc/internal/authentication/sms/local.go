package sms

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// LocalVerifier 在本地和测试环境使用固定验证码，避免调用外部短信服务。
// 生产环境禁用规则由 svc/sms_verifier.go 在创建本实现前强制检查。
type LocalVerifier struct {
	code string
}

func NewLocalVerifier(code string) (*LocalVerifier, error) {
	code = strings.TrimSpace(code)
	if len(code) < 4 || len(code) > 8 || strings.IndexFunc(code, func(value rune) bool {
		return !unicode.IsDigit(value)
	}) >= 0 {
		return nil, fmt.Errorf("local SMS code must contain 4 to 8 digits")
	}
	return &LocalVerifier{code: code}, nil
}

func (*LocalVerifier) SendLoginCode(context.Context, string) error {
	return nil
}

func (p *LocalVerifier) VerifyLoginCode(_ context.Context, _ string, code string) error {
	code = strings.TrimSpace(code)
	if code != p.code {
		return ErrInvalidCode
	}
	return nil
}
