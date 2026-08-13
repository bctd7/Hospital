package manager

import (
	"crypto/hmac"
	"crypto/sha256"
	"regexp"
	"strings"
)

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

// normalizePhone 将允许的手机号输入统一为 +86 E.164 形式。
func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	value = strings.TrimPrefix(value, "+86")
	if !mainlandPhone.MatchString(value) {
		return "", ErrInvalidPhone
	}
	return "+86" + value, nil
}

// maskPhone 只返回可展示的脱敏号码，数据库和日志不得保存登录时的明文手机号。
func maskPhone(normalized string) string {
	digits := strings.TrimPrefix(normalized, "+86")
	return digits[:3] + "****" + digits[7:]
}

// phoneFingerprint 生成可稳定检索但不可逆推出手机号的 HMAC 指纹。
func phoneFingerprint(key []byte, phone string) []byte {
	digest := hmac.New(sha256.New, key)
	_, _ = digest.Write([]byte(phone))
	return digest.Sum(nil)
}
