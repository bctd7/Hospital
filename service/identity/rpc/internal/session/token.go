package session

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

const refreshSecretSize = 32

// newRefreshToken 生成“sessionID.随机密钥”形式的不透明 Refresh Token，并返回用于服务端保存的哈希。
func newRefreshToken(sessionID string) (raw, hash string, err error) {
	secret := make([]byte, refreshSecretSize)
	if _, err = rand.Read(secret); err != nil {
		return "", "", err
	}
	raw = sessionID + "." + base64.RawURLEncoding.EncodeToString(secret)
	return raw, hashRefreshToken(raw), nil
}

// parseRefreshToken 校验 Token 格式和随机密钥长度，返回会话 ID 和待比对哈希；它不会读取账号或权限。
func parseRefreshToken(raw string) (sessionID, hash string, err error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return "", "", ErrInvalidRefreshToken
	}
	parsedID, err := uuid.Parse(parts[0])
	if err != nil || parsedID.String() != parts[0] {
		return "", "", ErrInvalidRefreshToken
	}
	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != refreshSecretSize {
		return "", "", ErrInvalidRefreshToken
	}
	return parts[0], hashRefreshToken(raw), nil
}

// hashRefreshToken 生成 Redis 中保存的固定长度哈希，避免持久化可直接使用的 Refresh Token 原文。
func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// tokenHashMatches 使用常量时间比较，降低通过比较耗时推测 Token 哈希的风险。
func tokenHashMatches(expected, actual string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
