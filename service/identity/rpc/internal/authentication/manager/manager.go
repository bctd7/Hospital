package manager

import (
	"errors"

	"hospital/service/identity/rpc/internal/authentication"
	"hospital/service/identity/rpc/internal/authentication/sms"
)

// Manager 编排短信验证码认证、账号解析和 Session 启动。
// 短信验证码的生成、发送和比对由 sms.Verifier 完成；Manager 不保存或比较验证码原文。
type Manager struct {
	store        authentication.Store
	verifier     sms.Verifier
	sessions     authentication.SessionStarter
	phoneKey     []byte
	retrySeconds int64
}

func NewManager(store authentication.Store, verifier sms.Verifier, sessions authentication.SessionStarter, phoneKey []byte, retrySeconds int64) (*Manager, error) {
	if store == nil || verifier == nil || sessions == nil {
		return nil, errors.New("phone authentication dependencies are required")
	}
	if len(phoneKey) < 32 {
		return nil, errors.New("identity phone lookup key must contain at least 32 bytes")
	}
	if retrySeconds <= 0 {
		retrySeconds = 60
	}
	return &Manager{
		store: store, verifier: verifier, sessions: sessions,
		phoneKey: append([]byte(nil), phoneKey...), retrySeconds: retrySeconds,
	}, nil
}
