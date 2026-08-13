// Package sms 适配短信验证码的发送和校验。
// 实现只处理验证码，不创建医院账号、不创建 Session，也不签发 Token。
package sms

import (
	"context"
	"errors"
)

var (
	ErrNotConfigured = errors.New("SMS verification is not configured")
	ErrUnavailable   = errors.New("SMS verification is unavailable")
	ErrInvalidCode   = errors.New("SMS verification code is invalid or expired")
	ErrRateLimited   = errors.New("SMS verification request is rate limited")
)

// Verifier 把验证码生成、发送、有效期和比对委托给短信平台实现。
// Identity 不接收也不持久化生产环境验证码原文。
type Verifier interface {
	SendLoginCode(ctx context.Context, phoneNumber string) error
	VerifyLoginCode(ctx context.Context, phoneNumber, code string) error
}

type UnconfiguredVerifier struct{}

func (UnconfiguredVerifier) SendLoginCode(context.Context, string) error {
	return ErrNotConfigured
}

func (UnconfiguredVerifier) VerifyLoginCode(context.Context, string, string) error {
	return ErrNotConfigured
}
